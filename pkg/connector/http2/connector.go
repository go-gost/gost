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
	md      metadata
	options connector.Options
}

func NewConnector(opts ...connector.Option) connector.Connector {
	options := connector.Options{}
	for _, opt := range opts {
		opt(&options)
	}

	return &http2Connector{
		options: options,
	}
}

func (c *http2Connector) Init(md md.Metadata) (err error) {
	return c.parseMetadata(md)
}

func (c *http2Connector) Connect(ctx context.Context, conn net.Conn, network, address string, opts ...connector.ConnectOption) (net.Conn, error) {
	log := c.options.Logger.WithFields(map[string]interface{}{
		"local":   conn.LocalAddr().String(),
		"remote":  conn.RemoteAddr().String(),
		"network": network,
		"address": address,
	})
	log.Infof("connect %s/%s", address, network)

	cc, ok := conn.(*http2_util.ClientConn)
	if !ok {
		err := errors.New("wrong connection type")
		log.Error(err)
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

	if user := c.options.Auth; user != nil {
		u := user.Username()
		p, _ := user.Password()
		req.Header.Set("Proxy-Authorization",
			"Basic "+base64.StdEncoding.EncodeToString([]byte(u+":"+p)))
	}

	if log.IsLevelEnabled(logger.DebugLevel) {
		dump, _ := httputil.DumpRequest(req, false)
		log.Debug(string(dump))
	}

	if c.md.connectTimeout > 0 {
		conn.SetDeadline(time.Now().Add(c.md.connectTimeout))
		defer conn.SetDeadline(time.Time{})
	}

	resp, err := cc.Client().Do(req.WithContext(ctx))
	if err != nil {
		log.Error(err)
		cc.Close()
		return nil, err
	}

	if log.IsLevelEnabled(logger.DebugLevel) {
		dump, _ := httputil.DumpResponse(resp, false)
		log.Debug(string(dump))
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		err = fmt.Errorf("%s", resp.Status)
		log.Error(err)
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
