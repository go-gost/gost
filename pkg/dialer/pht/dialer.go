package pht

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/go-gost/gost/pkg/dialer"
	pht_util "github.com/go-gost/gost/pkg/internal/util/pht"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
)

func init() {
	registry.DialerRegistry().Register("pht", NewDialer)
	registry.DialerRegistry().Register("phts", NewTLSDialer)
}

type phtDialer struct {
	tlsEnabled bool
	client     *pht_util.Client
	md         metadata
	logger     logger.Logger
	options    dialer.Options
}

func NewDialer(opts ...dialer.Option) dialer.Dialer {
	options := dialer.Options{}
	for _, opt := range opts {
		opt(&options)
	}

	return &phtDialer{
		logger:  options.Logger,
		options: options,
	}
}

func NewTLSDialer(opts ...dialer.Option) dialer.Dialer {
	options := dialer.Options{}
	for _, opt := range opts {
		opt(&options)
	}

	return &phtDialer{
		tlsEnabled: true,
		logger:     options.Logger,
		options:    options,
	}
}

func (d *phtDialer) Init(md md.Metadata) (err error) {
	if err = d.parseMetadata(md); err != nil {
		return
	}

	tr := &http.Transport{
		// Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	if d.tlsEnabled {
		tr.TLSClientConfig = d.options.TLSConfig
	}

	d.client = &pht_util.Client{
		Client: &http.Client{
			// Timeout:   60 * time.Second,
			Transport: tr,
		},
		AuthorizePath: d.md.authorizePath,
		PushPath:      d.md.pushPath,
		PullPath:      d.md.pullPath,
		TLSEnabled:    d.tlsEnabled,
		Logger:        d.options.Logger,
	}
	return nil
}

func (d *phtDialer) Dial(ctx context.Context, addr string, opts ...dialer.DialOption) (net.Conn, error) {
	return d.client.Dial(ctx, addr)
}
