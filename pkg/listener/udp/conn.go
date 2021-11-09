package udp

import (
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-gost/gost/pkg/internal/bufpool"
	"github.com/go-gost/gost/pkg/logger"
)

// conn is a server side connection for UDP client peer, it implements net.Conn and net.PacketConn.
type conn struct {
	net.PacketConn
	remoteAddr net.Addr
	rc         chan []byte // data receive queue
	idle       int32
	closed     chan struct{}
	closeMutex sync.Mutex
}

func newConn(c net.PacketConn, raddr net.Addr, queue int) *conn {
	return &conn{
		PacketConn: c,
		remoteAddr: raddr,
		rc:         make(chan []byte, queue),
		closed:     make(chan struct{}),
	}
}

func (c *conn) ReadFrom(b []byte) (n int, addr net.Addr, err error) {
	select {
	case bb := <-c.rc:
		n = copy(b, bb)
		c.SetIdle(false)
		bufpool.Put(bb)

	case <-c.closed:
		err = net.ErrClosed
		return
	}

	addr = c.remoteAddr

	return
}

func (c *conn) Read(b []byte) (n int, err error) {
	n, _, err = c.ReadFrom(b)
	return
}

func (c *conn) Write(b []byte) (n int, err error) {
	return c.WriteTo(b, c.remoteAddr)
}

func (c *conn) Close() error {
	c.closeMutex.Lock()
	defer c.closeMutex.Unlock()

	select {
	case <-c.closed:
	default:
		close(c.closed)
	}
	return nil
}

func (c *conn) RemoteAddr() net.Addr {
	return c.remoteAddr
}

func (c *conn) IsIdle() bool {
	return atomic.LoadInt32(&c.idle) > 0
}

func (c *conn) SetIdle(idle bool) {
	v := int32(0)
	if idle {
		v = 1
	}
	atomic.StoreInt32(&c.idle, v)
}

func (c *conn) Queue(b []byte) error {
	select {
	case c.rc <- b:
		return nil

	case <-c.closed:
		return net.ErrClosed

	default:
		return errors.New("recv queue is full")
	}
}

type connPool struct {
	m      sync.Map
	ttl    time.Duration
	closed chan struct{}
	logger logger.Logger
}

func newConnPool(ttl time.Duration) *connPool {
	p := &connPool{
		ttl:    ttl,
		closed: make(chan struct{}),
	}
	go p.idleCheck()
	return p
}

func (p *connPool) WithLogger(logger logger.Logger) *connPool {
	p.logger = logger
	return p
}

func (p *connPool) Get(key interface{}) (c *conn, ok bool) {
	v, ok := p.m.Load(key)
	if ok {
		c, ok = v.(*conn)
	}
	return
}

func (p *connPool) Set(key interface{}, c *conn) {
	p.m.Store(key, c)
}

func (p *connPool) Delete(key interface{}) {
	p.m.Delete(key)
}

func (p *connPool) Close() {
	select {
	case <-p.closed:
		return
	default:
	}

	close(p.closed)

	p.m.Range(func(k, v interface{}) bool {
		if c, ok := v.(*conn); ok && c != nil {
			c.Close()
		}
		return true
	})
}

func (p *connPool) idleCheck() {
	ticker := time.NewTicker(p.ttl)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			size := 0
			idles := 0
			p.m.Range(func(key, value interface{}) bool {
				c, ok := value.(*conn)
				if !ok || c == nil {
					p.Delete(key)
					return true
				}
				size++

				if c.IsIdle() {
					idles++
					p.Delete(key)
					c.Close()
					return true
				}

				c.SetIdle(true)

				return true
			})

			if idles > 0 {
				p.logger.Debugf("connection pool: size=%d, idle=%d", size, idles)
			}
		case <-p.closed:
			return
		}
	}
}
