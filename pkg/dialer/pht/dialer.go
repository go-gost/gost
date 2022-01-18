package pht

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
)

func init() {
	registry.RegisterDialer("pht", NewDialer)
	registry.RegisterDialer("phts", NewTLSDialer)
}

type phtDialer struct {
	tlsEnabled bool
	client     *http.Client
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

	d.client = &http.Client{
		Timeout:   60 * time.Second,
		Transport: tr,
	}
	return nil
}

func (d *phtDialer) Dial(ctx context.Context, addr string, opts ...dialer.DialOption) (net.Conn, error) {
	token, err := d.authorize(ctx, addr)
	if err != nil {
		d.logger.Error(err)
		return nil, err
	}

	c := &conn{
		cid:        token,
		addr:       addr,
		client:     d.client,
		tlsEnabled: d.tlsEnabled,
		rxc:        make(chan []byte, 128),
		closed:     make(chan struct{}),
		md:         d.md,
		logger:     d.logger,
	}
	go c.readLoop()

	return c, nil
}

func (d *phtDialer) authorize(ctx context.Context, addr string) (token string, err error) {
	var url string
	if d.tlsEnabled {
		url = fmt.Sprintf("https://%s%s", addr, d.md.authorizePath)
	} else {
		url = fmt.Sprintf("http://%s%s", addr, d.md.authorizePath)
	}
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
