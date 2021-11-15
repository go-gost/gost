package rtcp

import (
	"context"
	"fmt"
	"net"

	"github.com/go-gost/gosocks5"
	"github.com/go-gost/gost/pkg/chain"
	"github.com/go-gost/gost/pkg/internal/utils/mux"
	"github.com/go-gost/gost/pkg/internal/utils/socks"
)

func (l *rtcpListener) muxAccept() (net.Conn, error) {
	session, err := l.getSession()
	if err != nil {
		l.logger.Error(err)
		return nil, err
	}

	cc, err := session.Accept()
	if err != nil {
		session.Close()
		return nil, err
	}

	conn, err := l.getPeerConn(cc)
	if err != nil {
		l.logger.Error(err)
		cc.Close()
		return nil, err
	}

	l.logger.Debugf("peer %s accepted", conn.RemoteAddr())

	return conn, nil
}

func (l *rtcpListener) getPeerConn(conn net.Conn) (net.Conn, error) {
	// second reply, peer connected
	rep, err := gosocks5.ReadReply(conn)
	if err != nil {
		return nil, err
	}
	if rep.Rep != gosocks5.Succeeded {
		err = fmt.Errorf("peer connect failed")
		return nil, err
	}

	raddr, err := net.ResolveTCPAddr("tcp", rep.Addr.String())
	if err != nil {
		return nil, err
	}

	return &peerConn{
		Conn:       conn,
		localAddr:  l.laddr,
		remoteAddr: raddr,
	}, nil
}

func (l *rtcpListener) getSession() (s *mux.Session, err error) {
	l.sessionMux.Lock()
	defer l.sessionMux.Unlock()

	if l.session != nil && !l.session.IsClosed() {
		return l.session, nil
	}

	r := (&chain.Router{}).
		WithChain(l.chain).
		WithRetry(l.md.retryCount).
		WithLogger(l.logger)
	conn, err := r.Connect(context.Background())
	if err != nil {
		return nil, err
	}

	l.session, err = l.initSession(conn)
	if err != nil {
		conn.Close()
		return
	}

	return l.session, nil
}

func (l *rtcpListener) initSession(conn net.Conn) (*mux.Session, error) {
	addr := gosocks5.Addr{}
	addr.ParseFrom(l.addr)
	req := gosocks5.NewRequest(socks.CmdMuxBind, &addr)
	if err := req.Write(conn); err != nil {
		return nil, err
	}

	// first reply, bind status
	rep, err := gosocks5.ReadReply(conn)
	if err != nil {
		return nil, err
	}

	if rep.Rep != gosocks5.Succeeded {
		err = fmt.Errorf("bind on %s failed", l.addr)
		return nil, err
	}
	l.logger.Debugf("bind on %s OK", rep.Addr)

	return mux.ServerSession(conn)
}
