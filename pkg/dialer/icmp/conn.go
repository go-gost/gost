package quic

import (
	"context"
	"net"

	"github.com/lucas-clemente/quic-go"
)

type quicSession struct {
	session quic.Session
}

func (session *quicSession) GetConn() (*quicConn, error) {
	stream, err := session.session.OpenStreamSync(context.Background())
	if err != nil {
		return nil, err
	}
	return &quicConn{
		Stream: stream,
		laddr:  session.session.LocalAddr(),
		raddr:  session.session.RemoteAddr(),
	}, nil
}

func (session *quicSession) Close() error {
	return session.session.CloseWithError(quic.ApplicationErrorCode(0), "closed")
}

type quicConn struct {
	quic.Stream
	laddr net.Addr
	raddr net.Addr
}

func (c *quicConn) LocalAddr() net.Addr {
	return c.laddr
}

func (c *quicConn) RemoteAddr() net.Addr {
	return c.raddr
}
