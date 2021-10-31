package http

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/go-gost/gost/pkg/components/connector"
	md "github.com/go-gost/gost/pkg/components/metadata"
	"github.com/go-gost/gost/pkg/logger"
	"github.com/go-gost/gost/pkg/registry"
)

func init() {
	registry.RegiserConnector("http", NewConnector)
}

type httpConnector struct {
	md     metadata
	logger logger.Logger
}

func NewConnector(opts ...connector.Option) connector.Connector {
	options := &connector.Options{}
	for _, opt := range opts {
		opt(options)
	}

	return &httpConnector{
		logger: options.Logger,
	}
}

func (c *httpConnector) Init(md md.Metadata) (err error) {
	return c.parseMetadata(md)
}

func (c *httpConnector) Connect(ctx context.Context, conn net.Conn, network, address string, opts ...connector.ConnectOption) (net.Conn, error) {
	req := &http.Request{
		Method:     http.MethodConnect,
		URL:        &url.URL{Host: address},
		Host:       address,
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     make(http.Header),
	}
	if c.md.UserAgent != "" {
		req.Header.Set("User-Agent", c.md.UserAgent)
	}
	req.Header.Set("Proxy-Connection", "keep-alive")

	c.logger = c.logger.WithFields(map[string]interface{}{
		"src": conn.LocalAddr().String(),
		"dst": conn.RemoteAddr().String(),
	})

	if user := c.md.User; user != nil {
		u := user.Username()
		p, _ := user.Password()
		req.Header.Set("Proxy-Authorization",
			"Basic "+base64.StdEncoding.EncodeToString([]byte(u+":"+p)))
	}

	if c.logger.IsLevelEnabled(logger.DebugLevel) {
		dump, _ := httputil.DumpRequest(req, false)
		c.logger.Debug(string(dump))
	}

	req = req.WithContext(ctx)
	if err := req.Write(conn); err != nil {
		return nil, err
	}

	resp, err := http.ReadResponse(bufio.NewReader(conn), req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if c.logger.IsLevelEnabled(logger.DebugLevel) {
		dump, _ := httputil.DumpResponse(resp, false)
		c.logger.Debug(string(dump))
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s", resp.Status)
	}

	return conn, nil
}

func (c *httpConnector) parseMetadata(md md.Metadata) (err error) {
	c.md.UserAgent, _ = md.Get(userAgent).(string)
	if c.md.UserAgent == "" {
		c.md.UserAgent = defaultUserAgent
	}

	if v := md.GetString(auth); v != "" {
		ss := strings.SplitN(v, ":", 2)
		if len(ss) == 1 {
			c.md.User = url.User(ss[0])
		} else {
			c.md.User = url.UserPassword(ss[0], ss[1])
		}
	}

	return
}
