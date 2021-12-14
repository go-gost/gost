package mux

import (
	"crypto/tls"
	"net"

	"github.com/go-gost/gost/pkg/listener"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
	"github.com/xtaci/smux"
)

func init() {
	registry.RegisterListener("mtls", NewListener)
}

type mtlsListener struct {
	addr string
	md   metadata
	net.Listener
	connChan chan net.Conn
	errChan  chan error
	logger   logger.Logger
}

func NewListener(opts ...listener.Option) listener.Listener {
	options := &listener.Options{}
	for _, opt := range opts {
		opt(options)
	}
	return &mtlsListener{
		addr:   options.Addr,
		logger: options.Logger,
	}
}

func (l *mtlsListener) Init(md md.Metadata) (err error) {
	if err = l.parseMetadata(md); err != nil {
		return
	}

	ln, err := net.Listen("tcp", l.addr)
	if err != nil {
		return
	}
	l.Listener = tls.NewListener(ln, l.md.tlsConfig)

	queueSize := l.md.connQueueSize
	if queueSize <= 0 {
		queueSize = defaultQueueSize
	}
	l.connChan = make(chan net.Conn, queueSize)
	l.errChan = make(chan error, 1)

	go l.listenLoop()

	return
}

func (l *mtlsListener) listenLoop() {
	for {
		conn, err := l.Listener.Accept()
		if err != nil {
			l.errChan <- err
			close(l.errChan)
			return
		}
		go l.mux(conn)
	}
}

func (l *mtlsListener) mux(conn net.Conn) {
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

func (l *mtlsListener) Accept() (conn net.Conn, err error) {
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
