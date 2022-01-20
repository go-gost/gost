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

	"github.com/go-gost/gost/pkg/logger"
)

type Client struct {
	Client        *http.Client
	AuthorizePath string
	PushPath      string
	PullPath      string
	TLSEnabled    bool
	Logger        logger.Logger
}

func (c *Client) Dial(ctx context.Context, addr string) (net.Conn, error) {
	token, err := c.authorize(ctx, addr)
	if err != nil {
		c.Logger.Error(err)
		return nil, err
	}

	cn := &clientConn{
		client:    c.Client,
		rxc:       make(chan []byte, 128),
		closed:    make(chan struct{}),
		localAddr: &net.TCPAddr{},
		logger:    c.Logger,
	}
	cn.remoteAddr, _ = net.ResolveTCPAddr("tcp", addr)

	scheme := "http"
	if c.TLSEnabled {
		scheme = "https"
	}
	cn.pushURL = fmt.Sprintf("%s://%s%s?token=%s", scheme, addr, c.PushPath, token)
	cn.pullURL = fmt.Sprintf("%s://%s%s?token=%s", scheme, addr, c.PullPath, token)

	go cn.readLoop()

	return cn, nil
}

func (c *Client) authorize(ctx context.Context, addr string) (token string, err error) {
	var url string
	if c.TLSEnabled {
		url = fmt.Sprintf("https://%s%s", addr, c.AuthorizePath)
	} else {
		url = fmt.Sprintf("http://%s%s", addr, c.AuthorizePath)
	}
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return
	}

	if c.Logger.IsLevelEnabled(logger.DebugLevel) {
		dump, _ := httputil.DumpRequest(r, false)
		c.Logger.Debug(string(dump))
	}

	resp, err := c.Client.Do(r)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if c.Logger.IsLevelEnabled(logger.DebugLevel) {
		dump, _ := httputil.DumpResponse(resp, false)
		c.Logger.Debug(string(dump))
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
