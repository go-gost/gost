package http

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"

	"github.com/go-gost/gost/pkg/logger"
)

type obfsHTTPConn struct {
	net.Conn
	host           string
	rbuf           bytes.Buffer
	wbuf           bytes.Buffer
	headerDrained  bool
	handshaked     bool
	handshakeMutex sync.Mutex
	header         http.Header
	logger         logger.Logger
}

func (c *obfsHTTPConn) Handshake() (err error) {
	c.handshakeMutex.Lock()
	defer c.handshakeMutex.Unlock()

	if c.handshaked {
		return nil
	}

	err = c.handshake()
	if err != nil {
		return
	}

	c.handshaked = true
	return nil
}

func (c *obfsHTTPConn) handshake() (err error) {
	r := &http.Request{
		Method:     http.MethodGet,
		ProtoMajor: 1,
		ProtoMinor: 1,
		URL:        &url.URL{Scheme: "http", Host: c.host},
		Header:     c.header,
	}
	if r.Header == nil {
		r.Header = http.Header{}
	}
	r.Header.Set("Connection", "Upgrade")
	r.Header.Set("Upgrade", "websocket")
	key, _ := c.generateChallengeKey()
	r.Header.Set("Sec-WebSocket-Key", key)

	// cache the request header
	if err = r.Write(&c.wbuf); err != nil {
		return
	}

	if c.logger.IsLevelEnabled(logger.DebugLevel) {
		dump, _ := httputil.DumpRequest(r, false)
		c.logger.Debug(string(dump))
	}

	return nil
}

func (c *obfsHTTPConn) Read(b []byte) (n int, err error) {
	if err = c.Handshake(); err != nil {
		return
	}

	if err = c.drainHeader(); err != nil {
		return
	}

	if c.rbuf.Len() > 0 {
		return c.rbuf.Read(b)
	}
	return c.Conn.Read(b)
}

func (c *obfsHTTPConn) drainHeader() (err error) {
	if c.headerDrained {
		return
	}
	c.headerDrained = true

	br := bufio.NewReader(c.Conn)
	// drain and discard the response header
	var line string
	var buf bytes.Buffer
	for {
		line, err = br.ReadString('\n')
		if err != nil {
			return
		}
		buf.WriteString(line)
		if line == "\r\n" {
			break
		}
	}

	if c.logger.IsLevelEnabled(logger.DebugLevel) {
		c.logger.Debug(buf.String())
	}

	// cache the extra data for next read.
	var b []byte
	b, err = br.Peek(br.Buffered())
	if len(b) > 0 {
		_, err = c.rbuf.Write(b)
	}
	return
}

func (c *obfsHTTPConn) Write(b []byte) (n int, err error) {
	if err = c.Handshake(); err != nil {
		return
	}
	if c.wbuf.Len() > 0 {
		c.wbuf.Write(b) // append the data to the cached header
		_, err = c.wbuf.WriteTo(c.Conn)
		n = len(b) // exclude the header length
		return
	}
	return c.Conn.Write(b)
}

func (c *obfsHTTPConn) generateChallengeKey() (string, error) {
	p := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, p); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(p), nil
}
