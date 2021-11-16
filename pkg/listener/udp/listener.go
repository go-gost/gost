package udp

import (
	"net"

	"github.com/go-gost/gost/pkg/common/bufpool"
	"github.com/go-gost/gost/pkg/common/util/udp"
	"github.com/go-gost/gost/pkg/listener"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
)

func init() {
	registry.RegisterListener("udp", NewListener)
}

type udpListener struct {
	addr     string
	md       metadata
	conn     net.PacketConn
	cqueue   chan net.Conn
	errChan  chan error
	closed   chan struct{}
	connPool *udp.ConnPool
	logger   logger.Logger
}

func NewListener(opts ...listener.Option) listener.Listener {
	options := &listener.Options{}
	for _, opt := range opts {
		opt(options)
	}
	return &udpListener{
		addr:    options.Addr,
		errChan: make(chan error, 1),
		closed:  make(chan struct{}),
		logger:  options.Logger,
	}
}

func (l *udpListener) Init(md md.Metadata) (err error) {
	if err = l.parseMetadata(md); err != nil {
		return
	}

	laddr, err := net.ResolveUDPAddr("udp", l.addr)
	if err != nil {
		return
	}

	l.conn, err = net.ListenUDP("udp", laddr)
	if err != nil {
		return
	}

	l.cqueue = make(chan net.Conn, l.md.backlog)
	l.connPool = udp.NewConnPool(l.md.ttl).WithLogger(l.logger)

	go l.listenLoop()

	return
}

func (l *udpListener) Accept() (conn net.Conn, err error) {
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

func (l *udpListener) Close() error {
	select {
	case <-l.closed:
	default:
		close(l.closed)
		l.connPool.Close()
		return l.conn.Close()
	}

	return nil
}

func (l *udpListener) Addr() net.Addr {
	return l.conn.LocalAddr()
}

func (l *udpListener) listenLoop() {
	for {
		b := bufpool.Get(l.md.readBufferSize)

		n, raddr, err := l.conn.ReadFrom(b)
		if err != nil {
			l.errChan <- err
			close(l.errChan)
			return
		}

		c := l.getConn(raddr)
		if c == nil {
			bufpool.Put(b)
			continue
		}

		if err := c.WriteQueue(b[:n]); err != nil {
			l.logger.Warn("data discarded: ", err)
		}
	}
}

func (l *udpListener) getConn(addr net.Addr) *udp.Conn {
	c, ok := l.connPool.Get(addr.String())
	if !ok {
		c = udp.NewConn(l.conn, l.conn.LocalAddr(), addr, l.md.readQueueSize)
		select {
		case l.cqueue <- c:
			l.connPool.Set(addr.String(), c)
		default:
			c.Close()
			l.logger.Warnf("connection queue is full, client %s discarded", addr.String())
			return nil
		}
	}
	return c
}
