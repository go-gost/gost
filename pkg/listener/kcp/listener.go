package kcp

import (
	"net"
	"time"

	utils "github.com/go-gost/gost/pkg/internal/utils/kcp"
	"github.com/go-gost/gost/pkg/listener"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
	"github.com/xtaci/kcp-go/v5"
	"github.com/xtaci/smux"
	"github.com/xtaci/tcpraw"
)

func init() {
	registry.RegisterListener("kcp", NewListener)
}

type kcpListener struct {
	addr     string
	md       metadata
	ln       *kcp.Listener
	connChan chan net.Conn
	errChan  chan error
	logger   logger.Logger
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
	if config == nil {
		config = DefaultConfig
	}
	config.Init()

	var ln *kcp.Listener

	if config.TCP {
		var conn net.PacketConn
		conn, err = tcpraw.Listen("tcp", l.addr)
		if err != nil {
			return
		}
		ln, err = kcp.ServeConn(
			blockCrypt(config.Key, config.Crypt, Salt), config.DataShard, config.ParityShard, conn)
	} else {
		ln, err = kcp.ListenWithOptions(l.addr,
			blockCrypt(config.Key, config.Crypt, Salt), config.DataShard, config.ParityShard)
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
	l.connChan = make(chan net.Conn, l.md.connQueueSize)
	l.errChan = make(chan error, 1)

	go l.listenLoop()

	return
}

func (l *kcpListener) Accept() (conn net.Conn, err error) {
	var ok bool
	select {
	case conn = <-l.connChan:
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
		conn = utils.KCPCompStreamConn(conn)
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
			l.logger.Error("accept stream:", err)
			return
		}

		select {
		case l.connChan <- stream:
		case <-stream.GetDieCh():
			stream.Close()
		default:
			stream.Close()
			l.logger.Error("connection queue is full")
		}
	}
}

func (l *kcpListener) parseMetadata(md md.Metadata) (err error) {
	return
}
