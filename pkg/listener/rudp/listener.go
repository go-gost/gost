package rudp

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/go-gost/gosocks5"
	"github.com/go-gost/gost/pkg/chain"
	"github.com/go-gost/gost/pkg/common/bufpool"
	"github.com/go-gost/gost/pkg/common/util/socks"
	"github.com/go-gost/gost/pkg/common/util/udp"
	"github.com/go-gost/gost/pkg/listener"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
)

func init() {
	registry.RegisterListener("rudp", NewListener)
}

type rudpListener struct {
	addr     string
	laddr    *net.UDPAddr
	chain    *chain.Chain
	md       metadata
	cqueue   chan net.Conn
	closed   chan struct{}
	connPool *udp.ConnPool
	logger   logger.Logger
}

func NewListener(opts ...listener.Option) listener.Listener {
	options := &listener.Options{}
	for _, opt := range opts {
		opt(options)
	}
	return &rudpListener{
		addr:   options.Addr,
		closed: make(chan struct{}),
		logger: options.Logger,
	}
}

// implements listener.Chainable interface
func (l *rudpListener) Chain(chain *chain.Chain) {
	l.chain = chain
}

func (l *rudpListener) Init(md md.Metadata) (err error) {
	if err = l.parseMetadata(md); err != nil {
		return
	}

	laddr, err := net.ResolveUDPAddr("udp", l.addr)
	if err != nil {
		return
	}

	l.laddr = laddr
	l.cqueue = make(chan net.Conn, l.md.backlog)
	l.connPool = udp.NewConnPool(l.md.ttl).WithLogger(l.logger)

	go l.listenLoop()

	return
}

func (l *rudpListener) Accept() (conn net.Conn, err error) {
	select {
	case conn = <-l.cqueue:
		return
	case <-l.closed:
		return nil, listener.ErrClosed
	}
}

func (l *rudpListener) Close() error {
	select {
	case <-l.closed:
	default:
		close(l.closed)
		l.connPool.Close()
	}

	return nil
}

func (l *rudpListener) Addr() net.Addr {
	return l.laddr
}

func (l *rudpListener) listenLoop() {
	for {
		conn, err := l.connect()
		if err != nil {
			l.logger.Error(err)
			return
		}

		func() {
			defer conn.Close()

			for {
				b := bufpool.Get(l.md.readBufferSize)

				n, raddr, err := conn.ReadFrom(b)
				if err != nil {
					return
				}

				c := l.getConn(conn, raddr)
				if c == nil {
					bufpool.Put(b)
					continue
				}

				if err := c.WriteQueue(b[:n]); err != nil {
					l.logger.Warn("data discarded: ", err)
				}
			}
		}()
	}
}

func (l *rudpListener) connect() (conn net.PacketConn, err error) {
	var tempDelay time.Duration

	for {
		select {
		case <-l.closed:
			return nil, net.ErrClosed
		default:
		}

		conn, err = func() (net.PacketConn, error) {
			if l.chain.IsEmpty() {
				return net.ListenUDP("udp", l.laddr)
			}
			r := (&chain.Router{}).
				WithChain(l.chain).
				WithRetry(l.md.retryCount).
				WithLogger(l.logger)
			cc, err := r.Connect(context.Background())
			if err != nil {
				return nil, err
			}

			conn, err := l.initUDPTunnel(cc)
			if err != nil {
				cc.Close()
				return nil, err
			}
			return conn, err
		}()
		if err == nil {
			return
		}

		if tempDelay == 0 {
			tempDelay = 1000 * time.Millisecond
		} else {
			tempDelay *= 2
		}
		if max := 6 * time.Second; tempDelay > max {
			tempDelay = max
		}
		l.logger.Warnf("accept: %v, retrying in %v", err, tempDelay)
		time.Sleep(tempDelay)
	}
}

func (l *rudpListener) initUDPTunnel(conn net.Conn) (net.PacketConn, error) {
	socksAddr := gosocks5.Addr{}
	socksAddr.ParseFrom(l.laddr.String())
	req := gosocks5.NewRequest(socks.CmdUDPTun, &socksAddr)
	if err := req.Write(conn); err != nil {
		return nil, err
	}
	l.logger.Debug(req)

	reply, err := gosocks5.ReadReply(conn)
	if err != nil {
		return nil, err
	}
	l.logger.Debug(reply)

	if reply.Rep != gosocks5.Succeeded {
		return nil, fmt.Errorf("bind on %s failed", l.laddr)
	}

	baddr, err := net.ResolveUDPAddr("udp", reply.Addr.String())
	if err != nil {
		return nil, err
	}
	l.logger.Debugf("bind on %s OK", baddr)

	return socks.UDPTunClientPacketConn(conn), nil
}

func (l *rudpListener) getConn(conn net.PacketConn, raddr net.Addr) *udp.Conn {
	c, ok := l.connPool.Get(raddr.String())
	if !ok {
		c = udp.NewConn(conn, l.laddr, raddr, l.md.readQueueSize)
		select {
		case l.cqueue <- c:
			l.connPool.Set(raddr.String(), c)
		default:
			c.Close()
			l.logger.Warnf("connection queue is full, client %s discarded", raddr.String())
			return nil
		}
	}
	return c
}
