package mux

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
