package http2

import (
	"context"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/go-gost/gost/pkg/dialer"
	http2_util "github.com/go-gost/gost/pkg/internal/http2"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
)

func init() {
	registry.RegisterDialer("http2", NewDialer)
}

type http2Dialer struct {
	md          metadata
	clients     map[string]*http.Client
	clientMutex sync.Mutex
	logger      logger.Logger
}

func NewDialer(opts ...dialer.Option) dialer.Dialer {
	options := &dialer.Options{}
	for _, opt := range opts {
		opt(options)
	}

	return &http2Dialer{
		clients: make(map[string]*http.Client),
		logger:  options.Logger,
	}
}

func (d *http2Dialer) Init(md md.Metadata) (err error) {
	if err = d.parseMetadata(md); err != nil {
		return
	}

	return nil
}

// IsMultiplex implements dialer.Multiplexer interface.
func (d *http2Dialer) IsMultiplex() bool {
	return true
}

func (d *http2Dialer) Dial(ctx context.Context, address string, opts ...dialer.DialOption) (net.Conn, error) {
	options := &dialer.DialOptions{}
	for _, opt := range opts {
		opt(options)
	}

	raddr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		d.logger.Error(err)
		return nil, err
	}

	d.clientMutex.Lock()
	defer d.clientMutex.Unlock()

	client, ok := d.clients[address]
	if !ok {
		client = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: d.md.tlsConfig,
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					return d.dial(ctx, network, addr, options)
				},
				ForceAttemptHTTP2:     true,
				MaxIdleConns:          100,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			},
		}
		d.clients[address] = client
	}

	return http2_util.NewClientConn(
		&net.TCPAddr{}, raddr,
		client,
		func() {
			d.clientMutex.Lock()
			defer d.clientMutex.Unlock()
			delete(d.clients, address)
		}), nil
}

func (d *http2Dialer) dial(ctx context.Context, network, addr string, opts *dialer.DialOptions) (net.Conn, error) {
	dial := opts.DialFunc
	if dial != nil {
		conn, err := dial(ctx, addr)
		if err != nil {
			d.logger.Error(err)
		} else {
			d.logger.WithFields(map[string]interface{}{
				"src": conn.LocalAddr().String(),
				"dst": addr,
			}).Debug("dial with dial func")
		}
		return conn, err
	}

	var netd net.Dialer
	conn, err := netd.DialContext(ctx, network, addr)
	if err != nil {
		d.logger.Error(err)
	} else {
		d.logger.WithFields(map[string]interface{}{
			"src": conn.LocalAddr().String(),
			"dst": addr,
		}).Debugf("dial direct %s/%s", addr, network)
	}
	return conn, err
}
