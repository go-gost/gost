package ss

import (
	"bytes"
	"net"

	"github.com/go-gost/gosocks5"
	"github.com/go-gost/gost/pkg/internal/bufpool"
)

var (
	DefaultBufferSize = 4096
)

var (
	_ net.PacketConn = (*UDPConn)(nil)
	_ net.Conn       = (*UDPConn)(nil)
)

type UDPConn struct {
	net.PacketConn
	raddr      net.Addr
	taddr      net.Addr
	bufferSize int
}

func UDPClientConn(c net.PacketConn, remoteAddr, targetAddr net.Addr, bufferSize int) *UDPConn {
	return &UDPConn{
		PacketConn: c,
		raddr:      remoteAddr,
		taddr:      targetAddr,
		bufferSize: bufferSize,
	}
}

func UDPServerConn(c net.PacketConn, remoteAddr net.Addr, bufferSize int) *UDPConn {
	return &UDPConn{
		PacketConn: c,
		raddr:      remoteAddr,
		bufferSize: bufferSize,
	}
}

func (c *UDPConn) ReadFrom(b []byte) (n int, addr net.Addr, err error) {
	rbuf := bufpool.Get(c.bufferSize)
	defer bufpool.Put(rbuf)

	n, _, err = c.PacketConn.ReadFrom(rbuf)
	if err != nil {
		return
	}

	saddr := gosocks5.Addr{}
	addrLen, err := saddr.ReadFrom(bytes.NewReader(rbuf[:n]))
	if err != nil {
		return
	}

	n = copy(b, rbuf[addrLen:n])
	addr, err = net.ResolveUDPAddr("udp", saddr.String())

	return
}

func (c *UDPConn) Read(b []byte) (n int, err error) {
	n, _, err = c.ReadFrom(b)
	return
}

func (c *UDPConn) WriteTo(b []byte, addr net.Addr) (n int, err error) {
	wbuf := bufpool.Get(c.bufferSize)
	defer bufpool.Put(wbuf)

	socksAddr := gosocks5.Addr{}
	if err = socksAddr.ParseFrom(addr.String()); err != nil {
		return
	}

	addrLen, err := socksAddr.Encode(wbuf)
	if err != nil {
		return
	}

	n = copy(wbuf[addrLen:], b)
	_, err = c.PacketConn.WriteTo(wbuf[:addrLen+n], c.raddr)

	return
}

func (c *UDPConn) Write(b []byte) (n int, err error) {
	return c.WriteTo(b, c.taddr)
}

func (c *UDPConn) RemoteAddr() net.Addr {
	return c.raddr
}
