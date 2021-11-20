package v5

import (
	"fmt"
	"io"
	"net"

	"github.com/go-gost/gosocks5"
	"github.com/go-gost/gost/pkg/common/bufpool"
	"github.com/go-gost/gost/pkg/common/util/mux"
	"github.com/go-gost/gost/pkg/common/util/udp"
	"github.com/go-gost/gost/pkg/logger"
)

type tcpAccepter struct {
	addr   net.Addr
	conn   net.Conn
	logger logger.Logger
	done   chan struct{}
}

func (p *tcpAccepter) Accept() (net.Conn, error) {
	select {
	case <-p.done:
		return nil, io.EOF
	default:
		close(p.done)
	}

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

func (p *tcpAccepter) Addr() net.Addr {
	return p.addr
}

func (p *tcpAccepter) Close() error {
	return p.conn.Close()
}

type tcpMuxAccepter struct {
	addr    net.Addr
	session *mux.Session
	logger  logger.Logger
}

func (p *tcpMuxAccepter) Accept() (net.Conn, error) {
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

func (p *tcpMuxAccepter) getPeerConn(conn net.Conn) (net.Conn, error) {
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

func (p *tcpMuxAccepter) Addr() net.Addr {
	return p.addr
}

func (p *tcpMuxAccepter) Close() error {
	return p.session.Close()
}

type udpAccepter struct {
	addr           net.Addr
	conn           net.PacketConn
	cqueue         chan net.Conn
	connPool       *udp.ConnPool
	readQueueSize  int
	readBufferSize int
	closed         chan struct{}
	logger         logger.Logger
}

func (p *udpAccepter) Accept() (conn net.Conn, err error) {
	select {
	case conn = <-p.cqueue:
		return
	case <-p.closed:
		return nil, net.ErrClosed
	}
}

func (p *udpAccepter) acceptLoop() {
	for {
		select {
		case <-p.closed:
			return
		default:
		}

		b := bufpool.Get(p.readBufferSize)

		n, raddr, err := p.conn.ReadFrom(b)
		if err != nil {
			return
		}

		c := p.getConn(raddr)
		if c == nil {
			bufpool.Put(b)
			continue
		}

		if err := c.WriteQueue(b[:n]); err != nil {
			p.logger.Warn("data discarded: ", err)
		}
	}
}

func (p *udpAccepter) Addr() net.Addr {
	return p.addr
}

func (p *udpAccepter) Close() error {
	select {
	case <-p.closed:
	default:
		close(p.closed)
		p.connPool.Close()
	}

	return nil
}

func (p *udpAccepter) getConn(raddr net.Addr) *udp.Conn {
	c, ok := p.connPool.Get(raddr.String())
	if !ok {
		c = udp.NewConn(p.conn, p.addr, raddr, p.readQueueSize)
		select {
		case p.cqueue <- c:
			p.connPool.Set(raddr.String(), c)
		default:
			c.Close()
			p.logger.Warnf("connection queue is full, client %s discarded", raddr)
			return nil
		}
	}
	return c
}
