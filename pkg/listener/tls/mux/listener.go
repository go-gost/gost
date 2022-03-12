package mux

import (
	"crypto/tls"
	"net"

	"github.com/go-gost/gost/pkg/common/admission"
	"github.com/go-gost/gost/pkg/common/metrics"
	"github.com/go-gost/gost/pkg/listener"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
	"github.com/xtaci/smux"
)

func init() {
	registry.ListenerRegistry().Register("mtls", NewListener)
}

type mtlsListener struct {
	net.Listener
	cqueue  chan net.Conn
	errChan chan error
	logger  logger.Logger
	md      metadata
	options listener.Options
}

func NewListener(opts ...listener.Option) listener.Listener {
	options := listener.Options{}
	for _, opt := range opts {
		opt(&options)
	}
	return &mtlsListener{
		logger:  options.Logger,
		options: options,
	}
}

func (l *mtlsListener) Init(md md.Metadata) (err error) {
	if err = l.parseMetadata(md); err != nil {
		return
	}

	ln, err := net.Listen("tcp", l.options.Addr)
	if err != nil {
		return
	}

	ln = metrics.WrapListener(l.options.Service, ln)
	ln = admission.WrapListener(l.options.Admission, ln)
	l.Listener = tls.NewListener(ln, l.options.TLSConfig)

	l.cqueue = make(chan net.Conn, l.md.backlog)
	l.errChan = make(chan error, 1)

	go l.listenLoop()

	return
}

func (l *mtlsListener) Accept() (conn net.Conn, err error) {
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
