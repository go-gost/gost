package http3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/go-gost/gost/pkg/dialer"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
	"github.com/lucas-clemente/quic-go/http3"
)

func init() {
	registry.RegisterDialer("http3", NewDialer)
}

type http3Dialer struct {
	client  *http.Client
	md      metadata
	logger  logger.Logger
	options dialer.Options
}

func NewDialer(opts ...dialer.Option) dialer.Dialer {
	options := dialer.Options{}
	for _, opt := range opts {
		opt(&options)
	}

	tr := &http3.RoundTripper{
		TLSClientConfig: options.TLSConfig,
	}
	client := &http.Client{
		Timeout:   60 * time.Second,
		Transport: tr,
	}
	return &http3Dialer{
		client:  client,
		logger:  options.Logger,
		options: options,
	}
}

func (d *http3Dialer) Init(md md.Metadata) (err error) {
	return d.parseMetadata(md)
}

func (d *http3Dialer) Dial(ctx context.Context, addr string, opts ...dialer.DialOption) (net.Conn, error) {
	token, err := d.authorize(ctx, addr)
	if err != nil {
		d.logger.Error(err)
		return nil, err
	}

	c := &conn{
		cid:    token,
		addr:   addr,
		client: d.client,
		rxc:    make(chan []byte, 128),
		closed: make(chan struct{}),
		md:     d.md,
		logger: d.logger,
	}
	go c.readLoop()

	return c, nil
}

func (d *http3Dialer) authorize(ctx context.Context, addr string) (token string, err error) {
	url := fmt.Sprintf("https://%s%s", addr, d.md.authorizePath)
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return
	}

	if d.logger.IsLevelEnabled(logger.DebugLevel) {
		dump, _ := httputil.DumpRequest(r, false)
		d.logger.Debug(string(dump))
	}

	resp, err := d.client.Do(r)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if d.logger.IsLevelEnabled(logger.DebugLevel) {
		dump, _ := httputil.DumpResponse(resp, false)
		d.logger.Debug(string(dump))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	if strings.HasPrefix(string(data), "token=") {
		token = strings.TrimPrefix(string(data), "token=")
	}
	if token == "" {
		err = errors.New("authorize failed")
	}
	return
}
