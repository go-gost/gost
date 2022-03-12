package ws

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httputil"

	"github.com/go-gost/gost/pkg/common/admission"
	"github.com/go-gost/gost/pkg/common/metrics"
	ws_util "github.com/go-gost/gost/pkg/internal/util/ws"
	"github.com/go-gost/gost/pkg/listener"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
	"github.com/gorilla/websocket"
)

func init() {
	registry.ListenerRegistry().Register("ws", NewListener)
	registry.ListenerRegistry().Register("wss", NewTLSListener)
}

type wsListener struct {
	addr       net.Addr
	upgrader   *websocket.Upgrader
	srv        *http.Server
	tlsEnabled bool
	cqueue     chan net.Conn
	errChan    chan error
	logger     logger.Logger
	md         metadata
	options    listener.Options
}

func NewListener(opts ...listener.Option) listener.Listener {
	options := listener.Options{}
	for _, opt := range opts {
		opt(&options)
	}
	return &wsListener{
		logger:  options.Logger,
		options: options,
	}
}

func NewTLSListener(opts ...listener.Option) listener.Listener {
	options := listener.Options{}
	for _, opt := range opts {
		opt(&options)
	}
	return &wsListener{
		tlsEnabled: true,
		logger:     options.Logger,
		options:    options,
	}
}

func (l *wsListener) Init(md md.Metadata) (err error) {
	if err = l.parseMetadata(md); err != nil {
		return
	}

	l.upgrader = &websocket.Upgrader{
		HandshakeTimeout:  l.md.handshakeTimeout,
		ReadBufferSize:    l.md.readBufferSize,
		WriteBufferSize:   l.md.writeBufferSize,
		EnableCompression: l.md.enableCompression,
		CheckOrigin:       func(r *http.Request) bool { return true },
	}

	mux := http.NewServeMux()
	mux.Handle(l.md.path, http.HandlerFunc(l.upgrade))
	l.srv = &http.Server{
		Addr:              l.options.Addr,
		Handler:           mux,
		ReadHeaderTimeout: l.md.readHeaderTimeout,
	}

	l.cqueue = make(chan net.Conn, l.md.backlog)
	l.errChan = make(chan error, 1)

	ln, err := net.Listen("tcp", l.options.Addr)
	if err != nil {
		return
	}
	ln = metrics.WrapListener(l.options.Service, ln)
	ln = admission.WrapListener(l.options.Admission, ln)

	if l.tlsEnabled {
		ln = tls.NewListener(ln, l.options.TLSConfig)
	}

	l.addr = ln.Addr()

	go func() {
		err := l.srv.Serve(ln)
		if err != nil {
			l.errChan <- err
		}
		close(l.errChan)
	}()

	return
}

func (l *wsListener) Accept() (conn net.Conn, err error) {
	var ok bool
	select {
	case conn = <-l.cqueue:
	case err, ok = <-l.errChan:
		if !ok {
			err = listener.ErrClosed
		}
	}
	return
}

func (l *wsListener) Close() error {
	return l.srv.Close()
}

func (l *wsListener) Addr() net.Addr {
	return l.addr
}

func (l *wsListener) upgrade(w http.ResponseWriter, r *http.Request) {
	if l.logger.IsLevelEnabled(logger.DebugLevel) {
		log := l.logger.WithFields(map[string]any{
			"local":  l.addr.String(),
			"remote": r.RemoteAddr,
		})
		dump, _ := httputil.DumpRequest(r, false)
		log.Debug(string(dump))
	}

	conn, err := l.upgrader.Upgrade(w, r, l.md.header)
	if err != nil {
		l.logger.Error(err)
		return
	}

	select {
	case l.cqueue <- ws_util.Conn(conn):
	default:
		conn.Close()
		l.logger.Warnf("connection queue is full, client %s discarded", conn.RemoteAddr())
	}
}
