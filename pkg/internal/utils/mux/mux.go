package mux

import (
	"net"

	smux "github.com/xtaci/smux"
)

type MuxSession struct {
	conn    net.Conn
	session *smux.Session
}

func NewMuxSession(conn net.Conn) (*MuxSession, error) {
	// Upgrade connection to multiplex stream.
	s, err := smux.Client(conn, smux.DefaultConfig())
	if err != nil {
		return nil, err
	}
	return &MuxSession{
		conn:    conn,
		session: s,
	}, nil
}

func (session *MuxSession) GetConn() (net.Conn, error) {
	stream, err := session.session.OpenStream()
	if err != nil {
		return nil, err
	}
	return &muxStreamConn{Conn: session.conn, stream: stream}, nil
}

func (session *MuxSession) Accept() (net.Conn, error) {
	stream, err := session.session.AcceptStream()
	if err != nil {
		return nil, err
	}
	return &muxStreamConn{Conn: session.conn, stream: stream}, nil
}

func (session *MuxSession) Close() error {
	if session.session == nil {
		return nil
	}
	return session.session.Close()
}

func (session *MuxSession) IsClosed() bool {
	if session.session == nil {
		return true
	}
	return session.session.IsClosed()
}

func (session *MuxSession) NumStreams() int {
	return session.session.NumStreams()
}

type muxStreamConn struct {
	net.Conn
	stream *smux.Stream
}

func (c *muxStreamConn) Read(b []byte) (n int, err error) {
	return c.stream.Read(b)
}

func (c *muxStreamConn) Write(b []byte) (n int, err error) {
	return c.stream.Write(b)
}

func (c *muxStreamConn) Close() error {
	return c.stream.Close()
}
