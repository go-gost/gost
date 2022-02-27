package kcp

import (
	"net"
	"time"

	kcp_util "github.com/go-gost/gost/pkg/common/util/kcp"
	"github.com/go-gost/gost/pkg/listener"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
	"github.com/xtaci/kcp-go/v5"
	"github.com/xtaci/smux"
	"github.com/xtaci/tcpraw"
)

func init() {
	registry.ListenerRegistry().Register("kcp", NewListener)
}

type kcpListener struct {
	addr    string
	ln      *kcp.Listener
	cqueue  chan net.Conn
	errChan chan error
	logger  logger.Logger
	md      metadata
}

func NewListener(opts ...listener.Option) listener.Listener {
	options := &listener.Options{}
	for _, opt := range opts {
		opt(options)
	}
	return &kcpListener{
		addr:   options.Addr,
		logger: options.Logger,
	}
}

func (l *kcpListener) Init(md md.Metadata) (err error) {
	if err = l.parseMetadata(md); err != nil {
		return
	}

	config := l.md.config
	config.Init()

	var ln *kcp.Listener

	if config.TCP {
		var conn net.PacketConn
		conn, err = tcpraw.Listen("tcp", l.addr)
		if err != nil {
			return
		}
		ln, err = kcp.ServeConn(
			kcp_util.BlockCrypt(config.Key, config.Crypt, kcp_util.DefaultSalt), config.DataShard, config.ParityShard, conn)
	} else {
		ln, err = kcp.ListenWithOptions(l.addr,
			kcp_util.BlockCrypt(config.Key, config.Crypt, kcp_util.DefaultSalt), config.DataShard, config.ParityShard)
	}
	if err != nil {
		return
	}

	if config.DSCP > 0 {
		if err = ln.SetDSCP(config.DSCP); err != nil {
			l.logger.Warn(err)
		}
	}
	if err = ln.SetReadBuffer(config.SockBuf); err != nil {
		l.logger.Warn(err)
	}
	if err = ln.SetWriteBuffer(config.SockBuf); err != nil {
		l.logger.Warn(err)
	}

	l.ln = ln
	l.cqueue = make(chan net.Conn, l.md.backlog)
	l.errChan = make(chan error, 1)

	go l.listenLoop()

	return
}

func (l *kcpListener) Accept() (conn net.Conn, err error) {
	var ok bool
	select {
	case conn = <-l.cqueue:
	case err, ok = <-l.errChan:
		if !ok {
			err = listener.ErrClosed
		}
	}
	return
}

func (l *kcpListener) Close() error {
	return l.ln.Close()
}

func (l *kcpListener) Addr() net.Addr {
	return l.ln.Addr()
}

func (l *kcpListener) listenLoop() {
	for {
		conn, err := l.ln.AcceptKCP()
		if err != nil {
			l.logger.Error("accept:", err)
			l.errChan <- err
			close(l.errChan)
			return
		}

		conn.SetStreamMode(true)
		conn.SetWriteDelay(false)
		conn.SetNoDelay(
			l.md.config.NoDelay,
			l.md.config.Interval,
			l.md.config.Resend,
			l.md.config.NoCongestion,
		)
		conn.SetMtu(l.md.config.MTU)
		conn.SetWindowSize(l.md.config.SndWnd, l.md.config.RcvWnd)
		conn.SetACKNoDelay(l.md.config.AckNodelay)
		go l.mux(conn)
	}
}

func (l *kcpListener) mux(conn net.Conn) {
	defer conn.Close()

	smuxConfig := smux.DefaultConfig()
	smuxConfig.MaxReceiveBuffer = l.md.config.SockBuf
	smuxConfig.KeepAliveInterval = time.Duration(l.md.config.KeepAlive) * time.Second

	if !l.md.config.NoComp {
		conn = kcp_util.CompStreamConn(conn)
	}

	mux, err := smux.Server(conn, smuxConfig)
	if err != nil {
		l.logger.Error(err)
		return
	}
	defer mux.Close()

	for {
		stream, err := mux.AcceptStream()
		if err != nil {
			l.logger.Error("accept stream: ", err)
			return
		}

		select {
		case l.cqueue <- stream:
		case <-stream.GetDieCh():
			stream.Close()
		default:
			stream.Close()
			l.logger.Warnf("connection queue is full, client %s discarded", stream.RemoteAddr())
		}
	}
}
