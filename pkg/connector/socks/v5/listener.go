package v5

import (
	"fmt"
	"net"

	"github.com/go-gost/gosocks5"
	"github.com/go-gost/gost/pkg/common/util/mux"
	"github.com/go-gost/gost/pkg/logger"
)

type tcpListener struct {
	addr   net.Addr
	conn   net.Conn
	logger logger.Logger
}

func (p *tcpListener) Accept() (net.Conn, error) {
	// second reply, peer connected
	rep, err := gosocks5.ReadReply(p.conn)
	if err != nil {
		return nil, err
	}
	p.logger.Debug(rep)

	if rep.Rep != gosocks5.Succeeded {
		return nil, fmt.Errorf("peer connect failed")
	}

	raddr, err := net.ResolveTCPAddr("tcp", rep.Addr.String())
	if err != nil {
		return nil, err
	}

	return &bindConn{
		Conn:       p.conn,
		localAddr:  p.addr,
		remoteAddr: raddr,
	}, nil
}

func (p *tcpListener) Addr() net.Addr {
	return p.addr
}

func (p *tcpListener) Close() error {
	return p.conn.Close()
}

type tcpMuxListener struct {
	addr    net.Addr
	session *mux.Session
	logger  logger.Logger
}

func (p *tcpMuxListener) Accept() (net.Conn, error) {
	cc, err := p.session.Accept()
	if err != nil {
		return nil, err
	}

	conn, err := p.getPeerConn(cc)
	if err != nil {
		cc.Close()
		return nil, err
	}

	return conn, nil
}

func (p *tcpMuxListener) getPeerConn(conn net.Conn) (net.Conn, error) {
	// second reply, peer connected
	rep, err := gosocks5.ReadReply(conn)
	if err != nil {
		return nil, err
	}
	p.logger.Debug(rep)

	if rep.Rep != gosocks5.Succeeded {
		err = fmt.Errorf("peer connect failed")
		return nil, err
	}

	raddr, err := net.ResolveTCPAddr("tcp", rep.Addr.String())
	if err != nil {
		return nil, err
	}

	return &bindConn{
		Conn:       conn,
		localAddr:  p.addr,
		remoteAddr: raddr,
	}, nil
}

func (p *tcpMuxListener) Addr() net.Addr {
	return p.addr
}

func (p *tcpMuxListener) Close() error {
	return p.session.Close()
}
