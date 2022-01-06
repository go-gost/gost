package udp

import (
	"net"

	"github.com/LiamHaworth/go-tproxy"
	"github.com/go-gost/gost/pkg/common/bufpool"
)

func (l *redirectListener) listenUDP(addr *net.UDPAddr) (*net.UDPConn, error) {
	return tproxy.ListenUDP("udp", addr)
}

func (l *redirectListener) accept() (conn net.Conn, err error) {
	b := bufpool.Get(l.md.readBufferSize)

	n, raddr, dstAddr, err := tproxy.ReadFromUDP(l.ln, *b)
	if err != nil {
		l.logger.Error(err)
		return
	}

	l.logger.Infof("%s >> %s", raddr.String(), dstAddr.String())

	c, err := tproxy.DialUDP("udp", dstAddr, raddr)
	if err != nil {
		l.logger.Error(err)
		return
	}

	conn = &redirConn{
		Conn: c,
		buf:  (*b)[:n],
		ttl:  l.md.ttl,
	}
	return
}
