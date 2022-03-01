package ws

import (
	"context"
	"net"
	"net/url"
	"time"

	"github.com/go-gost/gost/pkg/dialer"
	ws_util "github.com/go-gost/gost/pkg/internal/util/ws"
	"github.com/go-gost/gost/pkg/logger"
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
	logger     logger.Logger
	md         metadata
	options    dialer.Options
}

func NewDialer(opts ...dialer.Option) dialer.Dialer {
	options := dialer.Options{}
	for _, opt := range opts {
		opt(&options)
	}

	return &wsDialer{
		logger:  options.Logger,
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
		logger:     options.Logger,
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

	netd := options.NetDialer
	if netd == nil {
		netd = dialer.DefaultNetDialer
	}
	conn, err := netd.Dial(ctx, "tcp", addr)
	if err != nil {
		d.logger.Error(err)
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

	c, resp, err := dialer.Dial(url.String(), d.md.header)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()

	return ws_util.Conn(c), nil
}
