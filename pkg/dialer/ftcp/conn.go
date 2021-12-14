package ftcp

import "net"

type fakeTCPConn struct {
	raddr net.Addr
	net.PacketConn
}

func (c *fakeTCPConn) Read(b []byte) (n int, err error) {
	n, _, err = c.ReadFrom(b)
	return
}

func (c *fakeTCPConn) Write(b []byte) (n int, err error) {
	return c.WriteTo(b, c.raddr)
}

func (c *fakeTCPConn) RemoteAddr() net.Addr {
	return c.raddr
}
