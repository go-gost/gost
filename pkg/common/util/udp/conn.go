package udp

import (
	"errors"
	"net"
	"sync"
	"sync/atomic"

	"github.com/go-gost/gost/pkg/common/bufpool"
)

// Conn is a server side connection for UDP client peer, it implements net.Conn and net.PacketConn.
type Conn struct {
	net.PacketConn
	localAddr  net.Addr
	remoteAddr net.Addr
	rc         chan []byte // data receive queue
	idle       int32       // indicate the connection is idle
	closed     chan struct{}
	closeMutex sync.Mutex
}

func NewConn(c net.PacketConn, localAddr, remoteAddr net.Addr, queueSize int) *Conn {
	return &Conn{
		PacketConn: c,
		localAddr:  localAddr,
		remoteAddr: remoteAddr,
		rc:         make(chan []byte, queueSize),
		closed:     make(chan struct{}),
	}
}

func (c *Conn) ReadFrom(b []byte) (n int, addr net.Addr, err error) {
	select {
	case bb := <-c.rc:
		n = copy(b, bb)
		c.SetIdle(false)
		bufpool.Put(&bb)

	case <-c.closed:
		err = net.ErrClosed
		return
	}

	addr = c.remoteAddr

	return
}

func (c *Conn) Read(b []byte) (n int, err error) {
	n, _, err = c.ReadFrom(b)
	return
}

func (c *Conn) Write(b []byte) (n int, err error) {
	return c.WriteTo(b, c.remoteAddr)
}

func (c *Conn) Close() error {
	c.closeMutex.Lock()
	defer c.closeMutex.Unlock()

	select {
	case <-c.closed:
	default:
		close(c.closed)
	}
	return nil
}

func (c *Conn) LocalAddr() net.Addr {
	return c.localAddr
}

func (c *Conn) RemoteAddr() net.Addr {
	return c.remoteAddr
}

func (c *Conn) IsIdle() bool {
	return atomic.LoadInt32(&c.idle) > 0
}

func (c *Conn) SetIdle(idle bool) {
	v := int32(0)
	if idle {
		v = 1
	}
	atomic.StoreInt32(&c.idle, v)
}

func (c *Conn) WriteQueue(b []byte) error {
	select {
	case c.rc <- b:
		return nil

	case <-c.closed:
		return net.ErrClosed

	default:
		return errors.New("recv queue is full")
	}
}
