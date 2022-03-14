package wrapper

import (
	"errors"
	"io"
	"net"
	"syscall"

	"github.com/go-gost/gost/v3/pkg/admission"
)

var (
	errUnsupport = errors.New("unsupported operation")
)

type packetConn struct {
	net.PacketConn
	admission admission.Admission
}

func WrapPacketConn(admission admission.Admission, pc net.PacketConn) net.PacketConn {
	if admission == nil {
		return pc
	}
	return &packetConn{
		PacketConn: pc,
		admission:  admission,
	}
}

func (c *packetConn) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	for {
		n, addr, err = c.PacketConn.ReadFrom(p)
		if err != nil {
			return
		}

		if c.admission != nil &&
			!c.admission.Admit(addr.String()) {
			continue
		}

		return
	}
}

type udpConn struct {
	net.PacketConn
	admission admission.Admission
}

func WrapUDPConn(admission admission.Admission, pc net.PacketConn) UDPConn {
	return &udpConn{
		PacketConn: pc,
		admission:  admission,
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
		return
	}
	err = errUnsupport
	return
}

func (c *udpConn) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	for {
		n, addr, err = c.PacketConn.ReadFrom(p)
		if err != nil {
			return
		}
		if c.admission != nil &&
			!c.admission.Admit(addr.String()) {
			continue
		}
		return
	}
}

func (c *udpConn) ReadFromUDP(b []byte) (n int, addr *net.UDPAddr, err error) {
	if nc, ok := c.PacketConn.(readUDP); ok {
		for {
			n, addr, err = nc.ReadFromUDP(b)
			if err != nil {
				return
			}
			if c.admission != nil &&
				!c.admission.Admit(addr.String()) {
				continue
			}
			return
		}
	}
	err = errUnsupport
	return
}

func (c *udpConn) ReadMsgUDP(b, oob []byte) (n, oobn, flags int, addr *net.UDPAddr, err error) {
	if nc, ok := c.PacketConn.(readUDP); ok {
		for {
			n, oobn, flags, addr, err = nc.ReadMsgUDP(b, oob)
			if err != nil {
				return
			}
			if c.admission != nil &&
				!c.admission.Admit(addr.String()) {
				continue
			}
			return
		}
	}
	err = errUnsupport
	return
}

func (c *udpConn) Write(b []byte) (n int, err error) {
	if nc, ok := c.PacketConn.(io.Writer); ok {
		n, err = nc.Write(b)
		return
	}
	err = errUnsupport
	return
}

func (c *udpConn) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	n, err = c.PacketConn.WriteTo(p, addr)
	return
}

func (c *udpConn) WriteToUDP(b []byte, addr *net.UDPAddr) (n int, err error) {
	if nc, ok := c.PacketConn.(writeUDP); ok {
		n, err = nc.WriteToUDP(b, addr)
		return
	}
	err = errUnsupport
	return
}

func (c *udpConn) WriteMsgUDP(b, oob []byte, addr *net.UDPAddr) (n, oobn int, err error) {
	if nc, ok := c.PacketConn.(writeUDP); ok {
		n, oobn, err = nc.WriteMsgUDP(b, oob, addr)
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
