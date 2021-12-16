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
	"time"

	"github.com/go-gost/gost/pkg/common/util/socks"
	"github.com/go-gost/gost/pkg/connector"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
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
	c.logger = c.logger.WithFields(map[string]interface{}{
		"local":   conn.LocalAddr().String(),
		"remote":  conn.RemoteAddr().String(),
		"network": network,
		"address": address,
	})
	c.logger.Infof("connect %s/%s", address, network)

	req := &http.Request{
		Method:     http.MethodConnect,
		URL:        &url.URL{Host: address},
		Host:       address,
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     make(http.Header),
	}
	req.Header.Set("Proxy-Connection", "keep-alive")

	if user := c.md.User; user != nil {
		u := user.Username()
		p, _ := user.Password()
		req.Header.Set("Proxy-Authorization",
			"Basic "+base64.StdEncoding.EncodeToString([]byte(u+":"+p)))
	}

	for k, v := range c.md.headers {
		req.Header.Set(k, v)
	}

	switch network {
	case "tcp", "tcp4", "tcp6":
		if _, ok := conn.(net.PacketConn); ok {
			err := fmt.Errorf("tcp over udp is unsupported")
			c.logger.Error(err)
			return nil, err
		}
	case "udp", "udp4", "udp6":
		req.Header.Set("X-Gost-Protocol", "udp")
	default:
		err := fmt.Errorf("network %s is unsupported", network)
		c.logger.Error(err)
		return nil, err
	}

	if c.logger.IsLevelEnabled(logger.DebugLevel) {
		dump, _ := httputil.DumpRequest(req, false)
		c.logger.Debug(string(dump))
	}

	if c.md.connectTimeout > 0 {
		conn.SetDeadline(time.Now().Add(c.md.connectTimeout))
		defer conn.SetDeadline(time.Time{})
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

	if network == "udp" {
		addr, _ := net.ResolveUDPAddr(network, address)
		return socks.UDPTunClientConn(conn, addr), nil
	}

	return conn, nil
}
