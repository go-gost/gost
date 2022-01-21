package http3

import (
	"context"
	"net"
	"net/http"

	"github.com/go-gost/gost/pkg/dialer"
	pht_util "github.com/go-gost/gost/pkg/internal/util/pht"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
	"github.com/lucas-clemente/quic-go/http3"
)

func init() {
	registry.RegisterDialer("http3", NewDialer)
}

type http3Dialer struct {
	client  *pht_util.Client
	md      metadata
	logger  logger.Logger
	options dialer.Options
}

func NewDialer(opts ...dialer.Option) dialer.Dialer {
	options := dialer.Options{}
	for _, opt := range opts {
		opt(&options)
	}

	return &http3Dialer{
		logger:  options.Logger,
		options: options,
	}
}

func (d *http3Dialer) Init(md md.Metadata) (err error) {
	if err = d.parseMetadata(md); err != nil {
		return
	}

	tr := &http3.RoundTripper{
		TLSClientConfig: d.options.TLSConfig,
	}
	d.client = &pht_util.Client{
		Client: &http.Client{
			// Timeout:   60 * time.Second,
			Transport: tr,
		},
		AuthorizePath: d.md.authorizePath,
		PushPath:      d.md.pushPath,
		PullPath:      d.md.pullPath,
		TLSEnabled:    true,
		Logger:        d.options.Logger,
	}
	return nil
}

func (d *http3Dialer) Dial(ctx context.Context, addr string, opts ...dialer.DialOption) (net.Conn, error) {
	return d.client.Dial(ctx, addr)
}
