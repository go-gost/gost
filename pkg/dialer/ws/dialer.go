package ws

import (
	"context"
	"net"
	"net/url"
	"time"

	"github.com/go-gost/gost/pkg/dialer"
	ws_util "github.com/go-gost/gost/pkg/internal/util/ws"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
	"github.com/gorilla/websocket"
)

func init() {
	registry.DialerRegistry().Register("ws", NewDialer)
	registry.DialerRegistry().Register("wss", NewTLSDialer)
}

type wsDialer struct {
	tlsEnabled bool
	md         metadata
	options    dialer.Options
}

func NewDialer(opts ...dialer.Option) dialer.Dialer {
	options := dialer.Options{}
	for _, opt := range opts {
		opt(&options)
	}

	return &wsDialer{
		options: options,
	}
}

func NewTLSDialer(opts ...dialer.Option) dialer.Dialer {
	options := dialer.Options{}
	for _, opt := range opts {
		opt(&options)
	}

	return &wsDialer{
		tlsEnabled: true,
		options:    options,
	}
}

func (d *wsDialer) Init(md md.Metadata) (err error) {
	return d.parseMetadata(md)
}

func (d *wsDialer) Dial(ctx context.Context, addr string, opts ...dialer.DialOption) (net.Conn, error) {
	var options dialer.DialOptions
	for _, opt := range opts {
		opt(&options)
	}

	conn, err := options.NetDialer.Dial(ctx, "tcp", addr)
	if err != nil {
		d.options.Logger.Error(err)
	}
	return conn, err
}

// Handshake implements dialer.Handshaker
func (d *wsDialer) Handshake(ctx context.Context, conn net.Conn, options ...dialer.HandshakeOption) (net.Conn, error) {
	opts := &dialer.HandshakeOptions{}
	for _, option := range options {
		option(opts)
	}

	if d.md.handshakeTimeout > 0 {
		conn.SetDeadline(time.Now().Add(d.md.handshakeTimeout))
		defer conn.SetDeadline(time.Time{})
	}

	host := d.md.host
	if host == "" {
		host = opts.Addr
	}

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

	c, resp, err := dialer.DialContext(ctx, url.String(), d.md.header)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()

	if d.md.keepAlive > 0 {
		c.SetReadDeadline(time.Now().Add(d.md.keepAlive * 2))
		c.SetPongHandler(func(string) error {
			c.SetReadDeadline(time.Now().Add(d.md.keepAlive * 2))
			d.options.Logger.Infof("pong: set read deadline: %v", d.md.keepAlive*2)
			return nil
		})
		go d.keepAlive(c)
	}

	return ws_util.Conn(c), nil
}

func (d *wsDialer) keepAlive(conn *websocket.Conn) {
	ticker := time.NewTicker(d.md.keepAlive)
	defer ticker.Stop()

	for range ticker.C {
		conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
			return
		}
		d.options.Logger.Infof("send ping")
	}
}
