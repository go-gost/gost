package udp

import "net"

type conn struct {
	*net.UDPConn
}

func (c *conn) WriteTo(b []byte, addr net.Addr) (int, error) {
	return c.UDPConn.Write(b)
}

func (c *conn) ReadFrom(b []byte) (n int, addr net.Addr, err error) {
	n, err = c.UDPConn.Read(b)
	addr = c.RemoteAddr()
	return
}
