package sni

import (
	"net"
)

type cacheConn struct {
	net.Conn
	buf []byte
}

func (c *cacheConn) Read(b []byte) (n int, err error) {
	if len(c.buf) > 0 {
		n = copy(b, c.buf)
		c.buf = c.buf[n:]
		return
	}

	return c.Conn.Read(b)
}
