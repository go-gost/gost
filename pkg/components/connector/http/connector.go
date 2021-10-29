package http

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"

	"github.com/go-gost/gost/pkg/components/connector"
	"github.com/go-gost/gost/pkg/logger"
	"github.com/go-gost/gost/pkg/registry"
)

func init() {
	registry.RegiserConnector("http", NewConnector)
}

type Connector struct {
	md     metadata
	logger logger.Logger
}

func NewConnector(opts ...connector.Option) connector.Connector {
	options := &connector.Options{}
	for _, opt := range opts {
		opt(options)
	}

	return &Connector{
		logger: options.Logger,
	}
}

func (c *Connector) Init(md connector.Metadata) (err error) {
	c.md, err = c.parseMetadata(md)
	if err != nil {
		return
	}
	return nil
}

func (c *Connector) Connect(ctx context.Context, conn net.Conn, network, address string, opts ...connector.ConnectOption) (net.Conn, error) {
	req := &http.Request{
		Method:     http.MethodConnect,
		URL:        &url.URL{Host: address},
		Host:       address,
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     make(http.Header),
	}
	if c.md.UserAgent != "" {
		log.Println(c.md.UserAgent)
		req.Header.Set("User-Agent", c.md.UserAgent)
	}
	req.Header.Set("Proxy-Connection", "keep-alive")

	if user := c.md.User; user != nil {
		u := user.Username()
		p, _ := user.Password()
		req.Header.Set("Proxy-Authorization",
			"Basic "+base64.StdEncoding.EncodeToString([]byte(u+":"+p)))
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

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s", resp.Status)
	}

	return conn, nil
}

func (c *Connector) parseMetadata(md connector.Metadata) (m metadata, err error) {
	if md == nil {
		md = connector.Metadata{}
	}
	m.UserAgent = md[userAgent]
	if m.UserAgent == "" {
		m.UserAgent = defaultUserAgent
	}

	if v, ok := md[username]; ok {
		m.User = url.UserPassword(v, md[password])
	}

	return
}
