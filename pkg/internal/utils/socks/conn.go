package socks

import (
	"bytes"
	"net"

	"github.com/go-gost/gosocks5"
	"github.com/go-gost/gost/pkg/internal/bufpool"
)

var (
	_ net.PacketConn = (*UDPTunConn)(nil)
	_ net.Conn       = (*UDPTunConn)(nil)

	_ net.PacketConn = (*UDPConn)(nil)
	_ net.Conn       = (*UDPConn)(nil)
)

type UDPTunConn struct {
	net.Conn
	taddr net.Addr
}

func UDPTunClientConn(c net.Conn, targetAddr net.Addr) *UDPTunConn {
	return &UDPTunConn{
		Conn:  c,
		taddr: targetAddr,
	}
}

func UDPTunServerConn(c net.Conn) *UDPTunConn {
	return &UDPTunConn{
		Conn: c,
	}
}

func (c *UDPTunConn) ReadFrom(b []byte) (n int, addr net.Addr, err error) {
	socksAddr := gosocks5.Addr{}
	header := gosocks5.UDPHeader{
		Addr: &socksAddr,
	}
	dgram := gosocks5.UDPDatagram{
		Header: &header,
		Data:   b,
	}
	_, err = dgram.ReadFrom(c.Conn)
	if err != nil {
		return
	}

	n = len(dgram.Data)
	if n > len(b) {
		n = copy(b, dgram.Data)
	}
	addr, err = net.ResolveUDPAddr("udp", socksAddr.String())

	return
}

func (c *UDPTunConn) Read(b []byte) (n int, err error) {
	n, _, err = c.ReadFrom(b)
	return
}

func (c *UDPTunConn) WriteTo(b []byte, addr net.Addr) (n int, err error) {
	socksAddr := gosocks5.Addr{}
	if err = socksAddr.ParseFrom(addr.String()); err != nil {
		return
	}

	header := gosocks5.UDPHeader{
		Addr: &socksAddr,
	}
	dgram := gosocks5.UDPDatagram{
		Header: &header,
		Data:   b,
	}
	dgram.Header.Rsv = uint16(len(dgram.Data))
	_, err = dgram.WriteTo(c.Conn)
	n = len(b)

	return
}

func (c *UDPTunConn) Write(b []byte) (n int, err error) {
	return c.WriteTo(b, c.taddr)
}

var (
	DefaultBufferSize = 4096
)

type UDPConn struct {
	net.PacketConn
	raddr      net.Addr
	taddr      net.Addr
	bufferSize int
}

func NewUDPConn(c net.PacketConn, bufferSize int) *UDPConn {
	return &UDPConn{
		PacketConn: c,
		bufferSize: bufferSize,
	}
}

// ReadFrom reads an UDP datagram.
// NOTE: for server side,
// the returned addr is the target address the client want to relay to.
func (c *UDPConn) ReadFrom(b []byte) (n int, addr net.Addr, err error) {
	rbuf := bufpool.Get(c.bufferSize)
	defer bufpool.Put(rbuf)

	n, c.raddr, err = c.PacketConn.ReadFrom(rbuf)
	if err != nil {
		return
	}

	socksAddr := gosocks5.Addr{}
	header := gosocks5.UDPHeader{
		Addr: &socksAddr,
	}
	hlen, err := header.ReadFrom(bytes.NewReader(rbuf[:n]))
	if err != nil {
		return
	}
	n = copy(b, rbuf[hlen:n])

	addr, err = net.ResolveUDPAddr("udp", socksAddr.String())
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

	header := gosocks5.UDPHeader{
		Addr: &socksAddr,
	}
	dgram := gosocks5.UDPDatagram{
		Header: &header,
		Data:   b,
	}

	buf := bytes.NewBuffer(wbuf[:0])
	_, err = dgram.WriteTo(buf)
	if err != nil {
		return
	}

	_, err = c.PacketConn.WriteTo(buf.Bytes(), c.raddr)
	n = len(b)

	return
}

func (c *UDPConn) Write(b []byte) (n int, err error) {
	return c.WriteTo(b, c.taddr)
}

func (c *UDPConn) RemoteAddr() net.Addr {
	return c.raddr
}
