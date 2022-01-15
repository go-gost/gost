package pht

import (
	"net"
)

// pht connection, wrapped up just like a net.Conn
type conn struct {
	net.Conn
	remoteAddr net.Addr
	localAddr  net.Addr
}

func (c *conn) LocalAddr() net.Addr {
	return c.localAddr
}

func (c *conn) RemoteAddr() net.Addr {
	return c.remoteAddr
}
