package rtcp

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/go-gost/gosocks5"
	"github.com/go-gost/gost/pkg/chain"
	"github.com/go-gost/gost/pkg/common/util/mux"
	"github.com/go-gost/gost/pkg/listener"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
)

func init() {
	registry.RegisterListener("rtcp", NewListener)
}

type rtcpListener struct {
	addr       string
	laddr      net.Addr
	chain      *chain.Chain
	md         metadata
	ln         net.Listener
	connChan   chan net.Conn
	session    *mux.Session
	sessionMux sync.Mutex
	logger     logger.Logger
	closed     chan struct{}
}

func NewListener(opts ...listener.Option) listener.Listener {
	options := &listener.Options{}
	for _, opt := range opts {
		opt(options)
	}
	return &rtcpListener{
		addr:   options.Addr,
		chain:  options.Chain,
		closed: make(chan struct{}),
		logger: options.Logger,
	}
}

func (l *rtcpListener) Init(md md.Metadata) (err error) {
	if err = l.parseMetadata(md); err != nil {
		return
	}

	laddr, err := net.ResolveTCPAddr("tcp", l.addr)
	if err != nil {
		return
	}

	l.laddr = laddr
	l.connChan = make(chan net.Conn, l.md.connQueueSize)

	if l.chain.IsEmpty() {
		l.ln, err = net.ListenTCP("tcp", laddr)
		return err
	}

	go l.listenLoop()

	return
}

func (l *rtcpListener) Addr() net.Addr {
	return l.laddr
}

func (l *rtcpListener) Close() error {
	if l.ln != nil {
		return l.ln.Close()
	}

	select {
	case <-l.closed:
	default:
		close(l.closed)
	}

	return nil
}

func (l *rtcpListener) Accept() (conn net.Conn, err error) {
	if l.ln != nil {
		return l.ln.Accept()
	}

	select {
	case conn = <-l.connChan:
	case <-l.closed:
		err = net.ErrClosed
	}

	return
}

func (l *rtcpListener) listenLoop() {
	var tempDelay time.Duration

	for {
		select {
		case <-l.closed:
			return
		default:
		}

		conn, err := l.accept()

		if err != nil {
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
			continue
		}

		tempDelay = 0

		select {
		case l.connChan <- conn:
		default:
			conn.Close()
			l.logger.Warnf("connection queue is full, client %s discarded", conn.RemoteAddr().String())
		}
	}
}

func (l *rtcpListener) accept() (net.Conn, error) {
	if l.md.enableMux {
		return l.muxAccept()
	}

	r := (&chain.Router{}).
		WithChain(l.chain).
		WithRetry(l.md.retryCount).
		WithLogger(l.logger)
	cc, err := r.Connect(context.Background())
	if err != nil {
		return nil, err
	}

	conn, err := l.waitPeer(cc)
	if err != nil {
		l.logger.Error(err)
		cc.Close()
		return nil, err
	}

	l.logger.Debugf("peer %s accepted", conn.RemoteAddr())

	return conn, nil
}

func (l *rtcpListener) waitPeer(conn net.Conn) (net.Conn, error) {
	addr := gosocks5.Addr{}
	addr.ParseFrom(l.addr)
	req := gosocks5.NewRequest(gosocks5.CmdBind, &addr)
	if err := req.Write(conn); err != nil {
		l.logger.Error(err)
		return nil, err
	}

	// first reply, bind status
	rep, err := gosocks5.ReadReply(conn)
	if err != nil {
		l.logger.Error(err)
		return nil, err
	}

	l.logger.Debug(rep)

	if rep.Rep != gosocks5.Succeeded {
		err = fmt.Errorf("bind on %s failed", l.addr)
		l.logger.Error(err)
		return nil, err
	}
	l.logger.Debugf("bind on %s OK", rep.Addr)

	// second reply, peer connected
	rep, err = gosocks5.ReadReply(conn)
	if err != nil {
		l.logger.Error(err)
		return nil, err
	}
	if rep.Rep != gosocks5.Succeeded {
		err = fmt.Errorf("peer connect failed")
		l.logger.Error(err)
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
