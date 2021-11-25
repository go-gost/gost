package relay

import (
	"fmt"
	"net"
	"strconv"

	"github.com/go-gost/gost/pkg/common/util/mux"
	"github.com/go-gost/gost/pkg/logger"
	"github.com/go-gost/relay"
)

type tcpListener struct {
	addr    net.Addr
	session *mux.Session
	logger  logger.Logger
}

func (p *tcpListener) Accept() (net.Conn, error) {
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

func (p *tcpListener) getPeerConn(conn net.Conn) (net.Conn, error) {
	// second reply, peer connected
	resp := relay.Response{}
	if _, err := resp.ReadFrom(conn); err != nil {
		return nil, err
	}

	if resp.Status != relay.StatusOK {
		err := fmt.Errorf("peer connect failed")
		return nil, err
	}

	var address string
	for _, f := range resp.Features {
		if f.Type() == relay.FeatureAddr {
			if fa, ok := f.(*relay.AddrFeature); ok {
				address = net.JoinHostPort(fa.Host, strconv.Itoa(int(fa.Port)))
			}
		}
	}

	raddr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		return nil, err
	}

	return &bindConn{
		Conn:       conn,
		localAddr:  p.addr,
		remoteAddr: raddr,
	}, nil
}

func (p *tcpListener) Addr() net.Addr {
	return p.addr
}

func (p *tcpListener) Close() error {
	return p.session.Close()
}
