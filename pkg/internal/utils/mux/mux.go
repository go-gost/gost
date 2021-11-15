package mux

import (
	"net"

	smux "github.com/xtaci/smux"
)

type Session struct {
	conn    net.Conn
	session *smux.Session
}

func ClientSession(conn net.Conn) (*Session, error) {
	s, err := smux.Client(conn, smux.DefaultConfig())
	if err != nil {
		return nil, err
	}
	return &Session{
		conn:    conn,
		session: s,
	}, nil
}

func ServerSession(conn net.Conn) (*Session, error) {
	s, err := smux.Server(conn, smux.DefaultConfig())
	if err != nil {
		return nil, err
	}
	return &Session{
		conn:    conn,
		session: s,
	}, nil
}

func (session *Session) GetConn() (net.Conn, error) {
	stream, err := session.session.OpenStream()
	if err != nil {
		return nil, err
	}
	return &StreamConn{Conn: session.conn, stream: stream}, nil
}

func (session *Session) Accept() (net.Conn, error) {
	stream, err := session.session.AcceptStream()
	if err != nil {
		return nil, err
	}
	return &StreamConn{Conn: session.conn, stream: stream}, nil
}

func (session *Session) Close() error {
	if session.session == nil {
		return nil
	}
	return session.session.Close()
}

func (session *Session) IsClosed() bool {
	if session.session == nil {
		return true
	}
	return session.session.IsClosed()
}

func (session *Session) NumStreams() int {
	return session.session.NumStreams()
}

type StreamConn struct {
	net.Conn
	stream *smux.Stream
}

func (c *StreamConn) Read(b []byte) (n int, err error) {
	return c.stream.Read(b)
}

func (c *StreamConn) Write(b []byte) (n int, err error) {
	return c.stream.Write(b)
}

func (c *StreamConn) Close() error {
	return c.stream.Close()
}
