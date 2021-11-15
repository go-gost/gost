package rtcp

import "net"

type peerConn struct {
	net.Conn
	localAddr  net.Addr
	remoteAddr net.Addr
}

func (c *peerConn) LocalAddr() net.Addr {
	return c.localAddr
}

func (c *peerConn) RemoteAddr() net.Addr {
	return c.remoteAddr
}
