package v5

import "net"

type bindConn struct {
	net.Conn
	localAddr  net.Addr
	remoteAddr net.Addr
}

func (c *bindConn) LocalAddr() net.Addr {
	return c.localAddr
}

func (c *bindConn) RemoteAddr() net.Addr {
	return c.remoteAddr
}
