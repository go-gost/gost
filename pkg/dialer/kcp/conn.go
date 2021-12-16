package kcp

import (
	"net"

	"github.com/xtaci/smux"
)

type muxSession struct {
	conn    net.Conn
	session *smux.Session
}

func (session *muxSession) GetConn() (net.Conn, error) {
	return session.session.OpenStream()
}

func (session *muxSession) Accept() (net.Conn, error) {
	return session.session.AcceptStream()
}

func (session *muxSession) Close() error {
	if session.session == nil {
		return nil
	}
	return session.session.Close()
}

func (session *muxSession) IsClosed() bool {
	if session.session == nil {
		return true
	}
	return session.session.IsClosed()
}

func (session *muxSession) NumStreams() int {
	return session.session.NumStreams()
}

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
