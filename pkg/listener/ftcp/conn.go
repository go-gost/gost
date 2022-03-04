package ftcp

import (
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// serverConn is a server side connection for UDP client peer, it implements net.Conn and net.PacketConn.
type serverConn struct {
	pc         net.PacketConn
	raddr      net.Addr
	rc         chan []byte // data receive queue
	fresh      int32
	closed     chan struct{}
	closeMutex sync.Mutex
	config     *serverConnConfig
}

type serverConnConfig struct {
	ttl     time.Duration
	qsize   int
	onClose func()
}

func newServerConn(conn net.PacketConn, raddr net.Addr, cfg *serverConnConfig) *serverConn {
	if conn == nil || raddr == nil {
		return nil
	}

	if cfg == nil {
		cfg = &serverConnConfig{}
	}
	c := &serverConn{
		pc:     conn,
		raddr:  raddr,
		rc:     make(chan []byte, cfg.qsize),
		closed: make(chan struct{}),
		config: cfg,
	}
	go c.ttlWait()
	return c
}

func (c *serverConn) send(b []byte) error {
	select {
	case c.rc <- b:
		return nil
	default:
		return errors.New("queue is full")
	}
}

func (c *serverConn) Read(b []byte) (n int, err error) {
	select {
	case bb := <-c.rc:
		n = copy(b, bb)
		atomic.StoreInt32(&c.fresh, 1)
	case <-c.closed:
		err = errors.New("read from closed connection")
		return
	}

	return
}

func (c *serverConn) Write(b []byte) (n int, err error) {
	return c.pc.WriteTo(b, c.raddr)
}

func (c *serverConn) Close() error {
	c.closeMutex.Lock()
	defer c.closeMutex.Unlock()

	select {
	case <-c.closed:
		return errors.New("connection is closed")
	default:
		if c.config.onClose != nil {
			c.config.onClose()
		}
		close(c.closed)
	}
	return nil
}

func (c *serverConn) LocalAddr() net.Addr {
	return c.pc.LocalAddr()
}

func (c *serverConn) RemoteAddr() net.Addr {
	return c.raddr
}

func (c *serverConn) SetDeadline(t time.Time) error {
	return c.pc.SetDeadline(t)
}

func (c *serverConn) SetReadDeadline(t time.Time) error {
	return c.pc.SetReadDeadline(t)
}

func (c *serverConn) SetWriteDeadline(t time.Time) error {
	return c.pc.SetWriteDeadline(t)
}

func (c *serverConn) ttlWait() {
	ticker := time.NewTicker(c.config.ttl)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if !atomic.CompareAndSwapInt32(&c.fresh, 1, 0) {
				c.Close()
				return
			}
		case <-c.closed:
			return
		}
	}
}
