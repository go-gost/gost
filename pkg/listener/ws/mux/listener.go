package mux

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httputil"

	ws_util "github.com/go-gost/gost/pkg/internal/util/ws"
	"github.com/go-gost/gost/pkg/listener"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
	"github.com/gorilla/websocket"
	"github.com/xtaci/smux"
)

func init() {
	registry.ListenerRegistry().Register("mws", NewListener)
	registry.ListenerRegistry().Register("mwss", NewTLSListener)
}

type mwsListener struct {
	addr       net.Addr
	upgrader   *websocket.Upgrader
	srv        *http.Server
	cqueue     chan net.Conn
	errChan    chan error
	tlsEnabled bool
	logger     logger.Logger
	md         metadata
	options    listener.Options
}

func NewListener(opts ...listener.Option) listener.Listener {
	options := listener.Options{}
	for _, opt := range opts {
		opt(&options)
	}
	return &mwsListener{
		logger:  options.Logger,
		options: options,
	}
}

func NewTLSListener(opts ...listener.Option) listener.Listener {
	options := listener.Options{}
	for _, opt := range opts {
		opt(&options)
	}
	return &mwsListener{
		tlsEnabled: true,
		logger:     options.Logger,
		options:    options,
	}
}

func (l *mwsListener) Init(md md.Metadata) (err error) {
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

	path := l.md.path
	if path == "" {
		path = defaultPath
	}
	mux := http.NewServeMux()
	mux.Handle(path, http.HandlerFunc(l.upgrade))
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

func (l *mwsListener) Accept() (conn net.Conn, err error) {
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

func (l *mwsListener) Close() error {
	return l.srv.Close()
}

func (l *mwsListener) Addr() net.Addr {
	return l.addr
}

func (l *mwsListener) upgrade(w http.ResponseWriter, r *http.Request) {
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

	l.mux(ws_util.Conn(conn))
}

func (l *mwsListener) mux(conn net.Conn) {
	defer conn.Close()

	smuxConfig := smux.DefaultConfig()
	smuxConfig.KeepAliveDisabled = l.md.muxKeepAliveDisabled
	if l.md.muxKeepAliveInterval > 0 {
		smuxConfig.KeepAliveInterval = l.md.muxKeepAliveInterval
	}
	if l.md.muxKeepAliveTimeout > 0 {
		smuxConfig.KeepAliveTimeout = l.md.muxKeepAliveTimeout
	}
	if l.md.muxMaxFrameSize > 0 {
		smuxConfig.MaxFrameSize = l.md.muxMaxFrameSize
	}
	if l.md.muxMaxReceiveBuffer > 0 {
		smuxConfig.MaxReceiveBuffer = l.md.muxMaxReceiveBuffer
	}
	if l.md.muxMaxStreamBuffer > 0 {
		smuxConfig.MaxStreamBuffer = l.md.muxMaxStreamBuffer
	}
	session, err := smux.Server(conn, smuxConfig)
	if err != nil {
		l.logger.Error(err)
		return
	}
	defer session.Close()

	for {
		stream, err := session.AcceptStream()
		if err != nil {
			l.logger.Error("accept stream: ", err)
			return
		}

		select {
		case l.cqueue <- stream:
		case <-stream.GetDieCh():
			stream.Close()
		default:
			stream.Close()
			l.logger.Warnf("connection queue is full, client %s discarded", stream.RemoteAddr())
		}
	}
}
