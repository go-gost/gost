package udp

import (
	"net"

	"github.com/go-gost/gost/pkg/internal/bufpool"
	"github.com/go-gost/gost/pkg/listener"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
)

func init() {
	registry.RegisterListener("udp", NewListener)
}

type udpListener struct {
	addr      string
	md        metadata
	conn      net.PacketConn
	connChan  chan net.Conn
	errChan   chan error
	closeChan chan struct{}
	connPool  *connPool
	logger    logger.Logger
}

func NewListener(opts ...listener.Option) listener.Listener {
	options := &listener.Options{}
	for _, opt := range opts {
		opt(options)
	}
	return &udpListener{
		addr:      options.Addr,
		errChan:   make(chan error, 1),
		closeChan: make(chan struct{}),
		logger:    options.Logger,
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

	l.connChan = make(chan net.Conn, l.md.connQueueSize)
	l.connPool = newConnPool(l.md.ttl).WithLogger(l.logger)

	go l.listenLoop()

	return
}

func (l *udpListener) Accept() (conn net.Conn, err error) {
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

func (l *udpListener) Close() error {
	select {
	case <-l.closeChan:
		return nil
	default:
		close(l.closeChan)
		l.connPool.Close()
		return l.conn.Close()
	}
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

		if err := c.Queue(b[:n]); err != nil {
			l.logger.Warn("data discarded: ", err)
		}
	}
}

func (l *udpListener) getConn(addr net.Addr) *conn {
	c, ok := l.connPool.Get(addr.String())
	if !ok {
		c = newConn(l.conn, addr, l.md.readQueueSize)
		select {
		case l.connChan <- c:
			l.connPool.Set(addr.String(), c)
		default:
			c.Close()
			l.logger.Warnf("connection queue is full, client %s discarded", addr.String())
			return nil
		}
	}
	return c
}

func (l *udpListener) parseMetadata(md md.Metadata) (err error) {
	l.md.ttl = md.GetDuration(ttl)
	if l.md.ttl <= 0 {
		l.md.ttl = defaultTTL
	}
	l.md.readBufferSize = md.GetInt(readBufferSize)
	if l.md.readBufferSize <= 0 {
		l.md.readBufferSize = defaultReadBufferSize
	}

	l.md.readQueueSize = md.GetInt(readQueueSize)
	if l.md.readQueueSize <= 0 {
		l.md.readQueueSize = defaultReadQueueSize
	}

	l.md.connQueueSize = md.GetInt(connQueueSize)
	if l.md.connQueueSize <= 0 {
		l.md.connQueueSize = defaultConnQueueSize
	}

	return
}
