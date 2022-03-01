package kcp

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"

	kcp_util "github.com/go-gost/gost/pkg/common/util/kcp"
	"github.com/go-gost/gost/pkg/dialer"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
	"github.com/xtaci/kcp-go/v5"
	"github.com/xtaci/smux"
	"github.com/xtaci/tcpraw"
)

func init() {
	registry.DialerRegistry().Register("kcp", NewDialer)
}

type kcpDialer struct {
	sessions     map[string]*muxSession
	sessionMutex sync.Mutex
	logger       logger.Logger
	md           metadata
}

func NewDialer(opts ...dialer.Option) dialer.Dialer {
	options := &dialer.Options{}
	for _, opt := range opts {
		opt(options)
	}

	return &kcpDialer{
		sessions: make(map[string]*muxSession),
		logger:   options.Logger,
	}
}

func (d *kcpDialer) Init(md md.Metadata) (err error) {
	if err = d.parseMetadata(md); err != nil {
		return
	}

	d.md.config.Init()

	return nil
}

// Multiplex implements dialer.Multiplexer interface.
func (d *kcpDialer) Multiplex() bool {
	return true
}

func (d *kcpDialer) Dial(ctx context.Context, addr string, opts ...dialer.DialOption) (conn net.Conn, err error) {
	d.sessionMutex.Lock()
	defer d.sessionMutex.Unlock()

	session, ok := d.sessions[addr]
	if session != nil && session.IsClosed() {
		delete(d.sessions, addr) // session is dead
		ok = false
	}
	if !ok {
		var options dialer.DialOptions
		for _, opt := range opts {
			opt(&options)
		}

		if d.md.config.TCP {
			raddr, err := net.ResolveUDPAddr("udp", addr)
			if err != nil {
				return nil, err
			}

			pc, err := tcpraw.Dial("tcp", addr)
			if err != nil {
				return nil, err
			}
			conn = &fakeTCPConn{
				raddr:      raddr,
				PacketConn: pc,
			}
		} else {
			netd := options.NetDialer
			if netd == nil {
				netd = dialer.DefaultNetDialer
			}
			conn, err = netd.Dial(ctx, "udp", addr)
			if err != nil {
				return nil, err
			}
		}
		session = &muxSession{conn: conn}
		d.sessions[addr] = session
	}

	return session.conn, err
}

// Handshake implements dialer.Handshaker
func (d *kcpDialer) Handshake(ctx context.Context, conn net.Conn, options ...dialer.HandshakeOption) (net.Conn, error) {
	opts := &dialer.HandshakeOptions{}
	for _, option := range options {
		option(opts)
	}

	d.sessionMutex.Lock()
	defer d.sessionMutex.Unlock()

	if d.md.handshakeTimeout > 0 {
		conn.SetDeadline(time.Now().Add(d.md.handshakeTimeout))
		defer conn.SetDeadline(time.Time{})
	}

	session, ok := d.sessions[opts.Addr]
	if session != nil && session.conn != conn {
		conn.Close()
		return nil, errors.New("kcp: unrecognized connection")
	}

	if !ok || session.session == nil {
		s, err := d.initSession(ctx, opts.Addr, conn)
		if err != nil {
			d.logger.Error(err)
			conn.Close()
			delete(d.sessions, opts.Addr)
			return nil, err
		}
		session = s
		d.sessions[opts.Addr] = session
	}
	cc, err := session.GetConn()
	if err != nil {
		session.Close()
		delete(d.sessions, opts.Addr)
		return nil, err
	}

	return cc, nil
}

func (d *kcpDialer) initSession(ctx context.Context, addr string, conn net.Conn) (*muxSession, error) {
	pc, ok := conn.(net.PacketConn)
	if !ok {
		return nil, errors.New("kcp: wrong connection type")
	}

	config := d.md.config

	kcpconn, err := kcp.NewConn(addr,
		kcp_util.BlockCrypt(config.Key, config.Crypt, kcp_util.DefaultSalt),
		config.DataShard, config.ParityShard, pc)
	if err != nil {
		return nil, err
	}

	kcpconn.SetStreamMode(true)
	kcpconn.SetWriteDelay(false)
	kcpconn.SetNoDelay(config.NoDelay, config.Interval, config.Resend, config.NoCongestion)
	kcpconn.SetWindowSize(config.SndWnd, config.RcvWnd)
	kcpconn.SetMtu(config.MTU)
	kcpconn.SetACKNoDelay(config.AckNodelay)

	if config.DSCP > 0 {
		if err := kcpconn.SetDSCP(config.DSCP); err != nil {
			d.logger.Warn("SetDSCP: ", err)
		}
	}
	if err := kcpconn.SetReadBuffer(config.SockBuf); err != nil {
		d.logger.Warn("SetReadBuffer: ", err)
	}
	if err := kcpconn.SetWriteBuffer(config.SockBuf); err != nil {
		d.logger.Warn("SetWriteBuffer: ", err)
	}

	// stream multiplex
	smuxConfig := smux.DefaultConfig()
	smuxConfig.MaxReceiveBuffer = config.SockBuf
	smuxConfig.KeepAliveInterval = time.Duration(config.KeepAlive) * time.Second
	var cc net.Conn = kcpconn
	if !config.NoComp {
		cc = kcp_util.CompStreamConn(kcpconn)
	}
	session, err := smux.Client(cc, smuxConfig)
	if err != nil {
		return nil, err
	}
	return &muxSession{conn: conn, session: session}, nil
}
