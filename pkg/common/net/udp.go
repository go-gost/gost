package net

import (
	"io"
	"net"
	"syscall"
)

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
