package udp

import (
	"net"
	"sync"
	"time"

	"github.com/go-gost/gost/pkg/common/bufpool"
)

type redirConn struct {
	net.Conn
	buf  []byte
	ttl  time.Duration
	once sync.Once
}

func (c *redirConn) Read(b []byte) (n int, err error) {
	if c.ttl > 0 {
		c.SetReadDeadline(time.Now().Add(c.ttl))
		defer c.SetReadDeadline(time.Time{})
	}

	c.once.Do(func() {
		n = copy(b, c.buf)
		bufpool.Put(c.buf)
		c.buf = nil
	})

	if n == 0 {
		n, err = c.Conn.Read(b)
	}
	return
}

func (c *redirConn) Write(b []byte) (n int, err error) {
	if c.ttl > 0 {
		c.SetWriteDeadline(time.Now().Add(c.ttl))
		defer c.SetWriteDeadline(time.Time{})
	}
	return c.Conn.Write(b)
}
