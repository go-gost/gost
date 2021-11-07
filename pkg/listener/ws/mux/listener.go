package mux

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
	"github.com/xtaci/smux"
)

func init() {
	registry.RegisterListener("mws", NewListener)
	registry.RegisterListener("mwss", NewListener)
}

type mwsListener struct {
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
	return &mwsListener{
		logger: options.Logger,
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

	l.connChan = make(chan net.Conn, l.md.connQueueSize)
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

func (l *mwsListener) Accept() (conn net.Conn, err error) {
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

func (l *mwsListener) Close() error {
	return l.srv.Close()
}

func (l *mwsListener) Addr() net.Addr {
	return l.addr
}

func (l *mwsListener) parseMetadata(md md.Metadata) (err error) {
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

func (l *mwsListener) upgrade(w http.ResponseWriter, r *http.Request) {
	conn, err := l.upgrader.Upgrade(w, r, l.md.responseHeader)
	if err != nil {
		l.logger.Error(err)
		return
	}

	l.mux(ws.WebsocketServerConn(conn))
}

func (l *mwsListener) mux(conn net.Conn) {
	smuxConfig := smux.DefaultConfig()
	smuxConfig.KeepAliveDisabled = l.md.muxKeepAliveDisabled
	if l.md.muxKeepAlivePeriod > 0 {
		smuxConfig.KeepAliveInterval = l.md.muxKeepAlivePeriod
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
			l.logger.Error("accept stream:", err)
			return
		}

		select {
		case l.connChan <- stream:
		case <-stream.GetDieCh():
			stream.Close()
		default:
			stream.Close()
			l.logger.Error("connection queue is full")
		}
	}
}
