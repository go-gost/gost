package metrics

import (
	"errors"
	"io"
	"net"
	"syscall"

	"github.com/go-gost/gost/v3/pkg/metrics"
)

var (
	errUnsupport = errors.New("unsupported operation")
)

// ServerConn is a server side Conn with metrics supported.
type serverConn struct {
	net.Conn
	service string
}

func WrapConn(service string, c net.Conn) net.Conn {
	return &serverConn{
		service: service,
		Conn:    c,
	}
}

func (c *serverConn) Read(b []byte) (n int, err error) {
	n, err = c.Conn.Read(b)
	metrics.InputBytes(c.service).Add(float64(n))
	return
}

func (c *serverConn) Write(b []byte) (n int, err error) {
	n, err = c.Conn.Write(b)
	metrics.OutputBytes(c.service).Add(float64(n))
	return
}

type packetConn struct {
	net.PacketConn
	service string
}

func WrapPacketConn(service string, pc net.PacketConn) net.PacketConn {
	return &packetConn{
		PacketConn: pc,
		service:    service,
	}
}

func (c *packetConn) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	n, addr, err = c.PacketConn.ReadFrom(p)
	metrics.InputBytes(c.service).Add(float64(n))
	return
}

func (c *packetConn) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	n, err = c.PacketConn.WriteTo(p, addr)
	metrics.OutputBytes(c.service).Add(float64(n))
	return
}

type udpConn struct {
	net.PacketConn
	service string
}

func WrapUDPConn(service string, pc net.PacketConn) UDPConn {
	return &udpConn{
		PacketConn: pc,
		service:    service,
	}
}

func (c *udpConn) RemoteAddr() net.Addr {
	if nc, ok := c.PacketConn.(remoteAddr); ok {
		return nc.RemoteAddr()
	}
	return nil
}

func (c *udpConn) SetReadBuffer(n int) error {
	if nc, ok := c.PacketConn.(setBuffer); ok {
		return nc.SetReadBuffer(n)
	}
	return errUnsupport
}

func (c *udpConn) SetWriteBuffer(n int) error {
	if nc, ok := c.PacketConn.(setBuffer); ok {
		return nc.SetWriteBuffer(n)
	}
	return errUnsupport
}

func (c *udpConn) Read(b []byte) (n int, err error) {
	if nc, ok := c.PacketConn.(io.Reader); ok {
		n, err = nc.Read(b)
		metrics.InputBytes(c.service).Add(float64(n))
		return
	}
	err = errUnsupport
	return
}

func (c *udpConn) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	n, addr, err = c.PacketConn.ReadFrom(p)
	metrics.InputBytes(c.service).Add(float64(n))
	return
}

func (c *udpConn) ReadFromUDP(b []byte) (n int, addr *net.UDPAddr, err error) {
	if nc, ok := c.PacketConn.(readUDP); ok {
		n, addr, err = nc.ReadFromUDP(b)
		metrics.InputBytes(c.service).Add(float64(n))
		return
	}
	err = errUnsupport
	return
}

func (c *udpConn) ReadMsgUDP(b, oob []byte) (n, oobn, flags int, addr *net.UDPAddr, err error) {
	if nc, ok := c.PacketConn.(readUDP); ok {
		n, oobn, flags, addr, err = nc.ReadMsgUDP(b, oob)
		metrics.InputBytes(c.service).Add(float64(n + oobn))
		return
	}
	err = errUnsupport
	return
}

func (c *udpConn) Write(b []byte) (n int, err error) {
	if nc, ok := c.PacketConn.(io.Writer); ok {
		n, err = nc.Write(b)
		metrics.OutputBytes(c.service).Add(float64(n))
		return
	}
	err = errUnsupport
	return
}

func (c *udpConn) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	n, err = c.PacketConn.WriteTo(p, addr)
	metrics.OutputBytes(c.service).Add(float64(n))
	return
}

func (c *udpConn) WriteToUDP(b []byte, addr *net.UDPAddr) (n int, err error) {
	if nc, ok := c.PacketConn.(writeUDP); ok {
		n, err = nc.WriteToUDP(b, addr)
		metrics.OutputBytes(c.service).Add(float64(n))
		return
	}
	err = errUnsupport
	return
}

func (c *udpConn) WriteMsgUDP(b, oob []byte, addr *net.UDPAddr) (n, oobn int, err error) {
	if nc, ok := c.PacketConn.(writeUDP); ok {
		n, oobn, err = nc.WriteMsgUDP(b, oob, addr)
		metrics.OutputBytes(c.service).Add(float64(n + oobn))
		return
	}
	err = errUnsupport
	return
}

func (c *udpConn) SyscallConn() (rc syscall.RawConn, err error) {
	if nc, ok := c.PacketConn.(syscallConn); ok {
		return nc.SyscallConn()
	}
	err = errUnsupport
	return
}

func (c *udpConn) SetDSCP(n int) error {
	if nc, ok := c.PacketConn.(setDSCP); ok {
		return nc.SetDSCP(n)
	}
	return nil
}

type UDPConn interface {
	net.PacketConn
	io.Reader
	io.Writer
	readUDP
	writeUDP
	setBuffer
	syscallConn
	remoteAddr
}

type setBuffer interface {
	SetReadBuffer(bytes int) error
	SetWriteBuffer(bytes int) error
}

type readUDP interface {
	ReadFromUDP(b []byte) (n int, addr *net.UDPAddr, err error)
	ReadMsgUDP(b, oob []byte) (n, oobn, flags int, addr *net.UDPAddr, err error)
}

type writeUDP interface {
	WriteToUDP(b []byte, addr *net.UDPAddr) (int, error)
	WriteMsgUDP(b, oob []byte, addr *net.UDPAddr) (n, oobn int, err error)
}

type syscallConn interface {
	SyscallConn() (syscall.RawConn, error)
}

type remoteAddr interface {
	RemoteAddr() net.Addr
}

// tcpraw.TCPConn
type setDSCP interface {
	SetDSCP(int) error
}
