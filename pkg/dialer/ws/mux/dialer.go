package mux

import (
	"context"
	"errors"
	"net"
	"net/url"
	"sync"
	"time"

	"github.com/go-gost/gost/pkg/dialer"
	ws_util "github.com/go-gost/gost/pkg/internal/util/ws"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
	"github.com/gorilla/websocket"
	"github.com/xtaci/smux"
)

func init() {
	registry.DialerRegistry().Register("mws", NewDialer)
	registry.DialerRegistry().Register("mwss", NewTLSDialer)
}

type mwsDialer struct {
	sessions     map[string]*muxSession
	sessionMutex sync.Mutex
	tlsEnabled   bool
	md           metadata
	options      dialer.Options
}

func NewDialer(opts ...dialer.Option) dialer.Dialer {
	options := dialer.Options{}
	for _, opt := range opts {
		opt(&options)
	}

	return &mwsDialer{
		sessions: make(map[string]*muxSession),
		options:  options,
	}
}

func NewTLSDialer(opts ...dialer.Option) dialer.Dialer {
	options := dialer.Options{}
	for _, opt := range opts {
		opt(&options)
	}

	return &mwsDialer{
		tlsEnabled: true,
		sessions:   make(map[string]*muxSession),
		options:    options,
	}
}
func (d *mwsDialer) Init(md md.Metadata) (err error) {
	if err = d.parseMetadata(md); err != nil {
		return
	}

	return nil
}

// Multiplex implements dialer.Multiplexer interface.
func (d *mwsDialer) Multiplex() bool {
	return true
}

func (d *mwsDialer) Dial(ctx context.Context, addr string, opts ...dialer.DialOption) (conn net.Conn, err error) {
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

		conn, err = options.NetDialer.Dial(ctx, "tcp", addr)
		if err != nil {
			return
		}

		session = &muxSession{conn: conn}
		d.sessions[addr] = session
	}

	return session.conn, err
}

// Handshake implements dialer.Handshaker
func (d *mwsDialer) Handshake(ctx context.Context, conn net.Conn, options ...dialer.HandshakeOption) (net.Conn, error) {
	opts := &dialer.HandshakeOptions{}
	for _, option := range options {
		option(opts)
	}

	d.sessionMutex.Lock()
	defer d.sessionMutex.Unlock()

	session, ok := d.sessions[opts.Addr]
	if session != nil && session.conn != conn {
		conn.Close()
		return nil, errors.New("mtls: unrecognized connection")
	}

	if !ok || session.session == nil {
		host := d.md.host
		if host == "" {
			host = opts.Addr
		}
		s, err := d.initSession(ctx, host, conn)
		if err != nil {
			d.options.Logger.Error(err)
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

func (d *mwsDialer) initSession(ctx context.Context, host string, conn net.Conn) (*muxSession, error) {
	dialer := websocket.Dialer{
		HandshakeTimeout:  d.md.handshakeTimeout,
		ReadBufferSize:    d.md.readBufferSize,
		WriteBufferSize:   d.md.writeBufferSize,
		EnableCompression: d.md.enableCompression,
		NetDial: func(net, addr string) (net.Conn, error) {
			return conn, nil
		},
	}

	url := url.URL{Scheme: "ws", Host: host, Path: d.md.path}
	if d.tlsEnabled {
		url.Scheme = "wss"
		dialer.TLSClientConfig = d.options.TLSConfig
	}

	if d.md.handshakeTimeout > 0 {
		conn.SetReadDeadline(time.Now().Add(d.md.handshakeTimeout))
	}

	c, resp, err := dialer.DialContext(ctx, url.String(), d.md.header)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()

	if d.md.handshakeTimeout > 0 {
		conn.SetReadDeadline(time.Time{})
	}

	cc := ws_util.Conn(c)

	if d.md.keepAlive > 0 {
		c.SetReadDeadline(time.Now().Add(d.md.keepAlive * 2))
		c.SetPongHandler(func(string) error {
			c.SetReadDeadline(time.Now().Add(d.md.keepAlive * 2))
			return nil
		})
		go d.keepAlive(cc)
	}

	// stream multiplex
	smuxConfig := smux.DefaultConfig()
	smuxConfig.KeepAliveDisabled = d.md.muxKeepAliveDisabled
	if d.md.muxKeepAliveInterval > 0 {
		smuxConfig.KeepAliveInterval = d.md.muxKeepAliveInterval
	}
	if d.md.muxKeepAliveTimeout > 0 {
		smuxConfig.KeepAliveTimeout = d.md.muxKeepAliveTimeout
	}
	if d.md.muxMaxFrameSize > 0 {
		smuxConfig.MaxFrameSize = d.md.muxMaxFrameSize
	}
	if d.md.muxMaxReceiveBuffer > 0 {
		smuxConfig.MaxReceiveBuffer = d.md.muxMaxReceiveBuffer
	}
	if d.md.muxMaxStreamBuffer > 0 {
		smuxConfig.MaxStreamBuffer = d.md.muxMaxStreamBuffer
	}

	session, err := smux.Client(cc, smuxConfig)
	if err != nil {
		return nil, err
	}
	return &muxSession{conn: cc, session: session}, nil
}

func (d *mwsDialer) keepAlive(conn ws_util.WebsocketConn) {
	ticker := time.NewTicker(d.md.keepAlive)
	defer ticker.Stop()

	for range ticker.C {
		conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
			return
		}
	}
}
