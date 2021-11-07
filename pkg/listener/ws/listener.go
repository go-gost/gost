package ws

import (
	"crypto/tls"
	"net"
	"net/http"

	util_tls "github.com/go-gost/gost/pkg/internal/utils/tls"
	"github.com/go-gost/gost/pkg/internal/utils/ws"
	"github.com/go-gost/gost/pkg/listener"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
	"github.com/gorilla/websocket"
)

func init() {
	registry.RegisterListener("ws", NewListener)
	registry.RegisterListener("wss", NewListener)
}

type wsListener struct {
	saddr    string
	md       metadata
	addr     net.Addr
	upgrader *websocket.Upgrader
	srv      *http.Server
	connChan chan net.Conn
	errChan  chan error
	logger   logger.Logger
}

func NewListener(opts ...listener.Option) listener.Listener {
	options := &listener.Options{}
	for _, opt := range opts {
		opt(options)
	}
	return &wsListener{
		saddr:  options.Addr,
		logger: options.Logger,
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
		CheckOrigin:       func(r *http.Request) bool { return true },
		EnableCompression: l.md.enableCompression,
	}

	path := l.md.path
	if path == "" {
		path = defaultPath
	}
	mux := http.NewServeMux()
	mux.Handle(path, http.HandlerFunc(l.upgrade))
	l.srv = &http.Server{
		Addr:              l.saddr,
		TLSConfig:         l.md.tlsConfig,
		Handler:           mux,
		ReadHeaderTimeout: l.md.readHeaderTimeout,
	}

	queueSize := l.md.connQueueSize
	if queueSize <= 0 {
		queueSize = defaultQueueSize
	}
	l.connChan = make(chan net.Conn, queueSize)
	l.errChan = make(chan error, 1)

	ln, err := net.Listen("tcp", l.saddr)
	if err != nil {
		return
	}
	if l.md.tlsConfig != nil {
		ln = tls.NewListener(ln, l.md.tlsConfig)
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
	case conn = <-l.connChan:
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

func (l *wsListener) parseMetadata(md md.Metadata) (err error) {
	l.md.tlsConfig, err = util_tls.LoadTLSConfig(
		md.GetString(certFile),
		md.GetString(keyFile),
		md.GetString(caFile),
	)
	if err != nil {
		return
	}

	return
}

func (l *wsListener) upgrade(w http.ResponseWriter, r *http.Request) {
	conn, err := l.upgrader.Upgrade(w, r, l.md.responseHeader)
	if err != nil {
		l.logger.Error(err)
		return
	}

	select {
	case l.connChan <- ws.WebsocketServerConn(conn):
	default:
		conn.Close()
		l.logger.Warn("connection queue is full")
	}
}
