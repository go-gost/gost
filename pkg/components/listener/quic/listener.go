package quic

import (
	"context"
	"net"

	"github.com/go-gost/gost/pkg/components/internal/utils"
	"github.com/go-gost/gost/pkg/components/listener"
	md "github.com/go-gost/gost/pkg/components/metadata"
	"github.com/go-gost/gost/pkg/logger"
	"github.com/go-gost/gost/pkg/registry"
	"github.com/lucas-clemente/quic-go"
)

func init() {
	registry.RegisterListener("quic", NewListener)
}

type Listener struct {
	addr     string
	md       metadata
	ln       quic.Listener
	connChan chan net.Conn
	errChan  chan error
	logger   logger.Logger
}

func NewListener(opts ...listener.Option) listener.Listener {
	options := &listener.Options{}
	for _, opt := range opts {
		opt(options)
	}
	return &Listener{
		addr:   options.Addr,
		logger: options.Logger,
	}
}

func (l *Listener) Init(md md.Metadata) (err error) {
	if err = l.parseMetadata(md); err != nil {
		return
	}

	laddr, err := net.ResolveUDPAddr("udp", l.addr)
	if err != nil {
		return
	}

	var conn net.PacketConn
	conn, err = net.ListenUDP("udp", laddr)
	if err != nil {
		return
	}

	if l.md.cipherKey != nil {
		conn = utils.QUICCipherConn(conn, l.md.cipherKey)
	}

	config := &quic.Config{
		KeepAlive:            l.md.keepAlive,
		HandshakeIdleTimeout: l.md.HandshakeTimeout,
		MaxIdleTimeout:       l.md.MaxIdleTimeout,
	}

	ln, err := quic.Listen(conn, l.md.tlsConfig, config)
	if err != nil {
		return
	}

	l.ln = ln
	l.connChan = make(chan net.Conn, l.md.connQueueSize)
	l.errChan = make(chan error, 1)

	go l.listenLoop()

	return
}

func (l *Listener) Accept() (conn net.Conn, err error) {
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

func (l *Listener) Close() error {
	return l.ln.Close()
}

func (l *Listener) Addr() net.Addr {
	return l.ln.Addr()
}

func (l *Listener) listenLoop() {
	for {
		ctx := context.Background()
		session, err := l.ln.Accept(ctx)
		if err != nil {
			l.logger.Error("accept:", err)
			l.errChan <- err
			close(l.errChan)
			return
		}
		go l.mux(ctx, session)
	}
}

func (l *Listener) mux(ctx context.Context, session quic.Session) {
	defer session.CloseWithError(0, "")

	for {
		stream, err := session.AcceptStream(ctx)
		if err != nil {
			l.logger.Error("accept stream:", err)
			return
		}

		conn := utils.QUICConn(session, stream)
		select {
		case l.connChan <- conn:
		case <-stream.Context().Done():
			stream.Close()
		default:
			stream.Close()
			l.logger.Error("connection queue is full")
		}
	}
}

func (l *Listener) parseMetadata(md md.Metadata) (err error) {

	return
}
