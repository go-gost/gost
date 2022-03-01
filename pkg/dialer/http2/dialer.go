package http2

import (
	"context"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/go-gost/gost/pkg/dialer"
	http2_util "github.com/go-gost/gost/pkg/internal/util/http2"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
)

func init() {
	registry.DialerRegistry().Register("http2", NewDialer)
}

type http2Dialer struct {
	clients     map[string]*http.Client
	clientMutex sync.Mutex
	logger      logger.Logger
	md          metadata
	options     dialer.Options
}

func NewDialer(opts ...dialer.Option) dialer.Dialer {
	options := dialer.Options{}
	for _, opt := range opts {
		opt(&options)
	}

	return &http2Dialer{
		clients: make(map[string]*http.Client),
		logger:  options.Logger,
		options: options,
	}
}

func (d *http2Dialer) Init(md md.Metadata) (err error) {
	if err = d.parseMetadata(md); err != nil {
		return
	}

	return nil
}

// Multiplex implements dialer.Multiplexer interface.
func (d *http2Dialer) Multiplex() bool {
	return true
}

func (d *http2Dialer) Dial(ctx context.Context, address string, opts ...dialer.DialOption) (net.Conn, error) {
	raddr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		d.logger.Error(err)
		return nil, err
	}

	d.clientMutex.Lock()
	defer d.clientMutex.Unlock()

	client, ok := d.clients[address]
	if !ok {
		options := dialer.DialOptions{}
		for _, opt := range opts {
			opt(&options)
		}

		client = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: d.options.TLSConfig,
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					netd := options.NetDialer
					if netd == nil {
						netd = dialer.DefaultNetDialer
					}
					return netd.Dial(ctx, network, addr)
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
