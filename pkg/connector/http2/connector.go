package http2

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/go-gost/gost/pkg/connector"
	http2_util "github.com/go-gost/gost/pkg/internal/util/http2"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
)

func init() {
	registry.RegiserConnector("http2", NewConnector)
}

type http2Connector struct {
	user   *url.Userinfo
	md     metadata
	logger logger.Logger
}

func NewConnector(opts ...connector.Option) connector.Connector {
	options := &connector.Options{}
	for _, opt := range opts {
		opt(options)
	}

	return &http2Connector{
		user:   options.User,
		logger: options.Logger,
	}
}

func (c *http2Connector) Init(md md.Metadata) (err error) {
	return c.parseMetadata(md)
}

func (c *http2Connector) Connect(ctx context.Context, conn net.Conn, network, address string, opts ...connector.ConnectOption) (net.Conn, error) {
	c.logger = c.logger.WithFields(map[string]interface{}{
		"local":   conn.LocalAddr().String(),
		"remote":  conn.RemoteAddr().String(),
		"network": network,
		"address": address,
	})
	c.logger.Infof("connect %s/%s", address, network)

	cc, ok := conn.(*http2_util.ClientConn)
	if !ok {
		err := errors.New("wrong connection type")
		c.logger.Error(err)
		return nil, err
	}

	pr, pw := io.Pipe()
	req := &http.Request{
		Method:     http.MethodConnect,
		URL:        &url.URL{Scheme: "https", Host: conn.RemoteAddr().String()},
		Host:       address,
		ProtoMajor: 2,
		ProtoMinor: 0,
		Header:     make(http.Header),
		Body:       pr,
		// ContentLength: -1,
	}
	if c.md.UserAgent != "" {
		req.Header.Set("User-Agent", c.md.UserAgent)
	}

	if user := c.user; user != nil {
		u := user.Username()
		p, _ := user.Password()
		req.Header.Set("Proxy-Authorization",
			"Basic "+base64.StdEncoding.EncodeToString([]byte(u+":"+p)))
	}

	if c.logger.IsLevelEnabled(logger.DebugLevel) {
		dump, _ := httputil.DumpRequest(req, false)
		c.logger.Debug(string(dump))
	}

	if c.md.connectTimeout > 0 {
		conn.SetDeadline(time.Now().Add(c.md.connectTimeout))
		defer conn.SetDeadline(time.Time{})
	}

	resp, err := cc.Client().Do(req.WithContext(ctx))
	if err != nil {
		c.logger.Error(err)
		cc.Close()
		return nil, err
	}

	if c.logger.IsLevelEnabled(logger.DebugLevel) {
		dump, _ := httputil.DumpResponse(resp, false)
		c.logger.Debug(string(dump))
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		err = fmt.Errorf("%s", resp.Status)
		c.logger.Error(err)
		return nil, err
	}

	hc := &http2Conn{
		r:         resp.Body,
		w:         pw,
		localAddr: conn.RemoteAddr(),
	}

	hc.remoteAddr, _ = net.ResolveTCPAddr(network, address)

	return hc, nil
}
