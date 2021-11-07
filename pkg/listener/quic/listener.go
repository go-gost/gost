package quic

import (
	"context"
	"net"

	utils "github.com/go-gost/gost/pkg/internal/utils/quic"
	"github.com/go-gost/gost/pkg/listener"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
	"github.com/lucas-clemente/quic-go"
)

func init() {
	registry.RegisterListener("quic", NewListener)
}

type quicListener struct {
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
	return &quicListener{
		addr:   options.Addr,
		logger: options.Logger,
	}
}

func (l *quicListener) Init(md md.Metadata) (err error) {
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

func (l *quicListener) Accept() (conn net.Conn, err error) {
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

func (l *quicListener) Close() error {
	return l.ln.Close()
}

func (l *quicListener) Addr() net.Addr {
	return l.ln.Addr()
}

func (l *quicListener) listenLoop() {
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

func (l *quicListener) mux(ctx context.Context, session quic.Session) {
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

func (l *quicListener) parseMetadata(md md.Metadata) (err error) {

	return
}
