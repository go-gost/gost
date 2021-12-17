package http

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"sync"
	"time"

	"github.com/go-gost/gost/pkg/logger"
)

type obfsHTTPConn struct {
	net.Conn
	rbuf           bytes.Buffer
	wbuf           bytes.Buffer
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

	if err = c.handshake(); err != nil {
		return
	}

	c.handshaked = true
	return nil
}

func (c *obfsHTTPConn) handshake() (err error) {
	br := bufio.NewReader(c.Conn)
	r, err := http.ReadRequest(br)
	if err != nil {
		return
	}

	if c.logger.IsLevelEnabled(logger.DebugLevel) {
		dump, _ := httputil.DumpRequest(r, false)
		c.logger.Debug(string(dump))
	}

	if r.ContentLength > 0 {
		_, err = io.Copy(&c.rbuf, r.Body)
	} else {
		var b []byte
		b, err = br.Peek(br.Buffered())
		if len(b) > 0 {
			_, err = c.rbuf.Write(b)
		}
	}
	if err != nil {
		c.logger.Error(err)
		return
	}

	resp := http.Response{
		StatusCode: http.StatusOK,
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     c.header,
	}
	if resp.Header == nil {
		resp.Header = http.Header{}
	}
	resp.Header.Set("Date", time.Now().Format(time.RFC1123))

	if r.Method != http.MethodGet || r.Header.Get("Upgrade") != "websocket" {
		resp.StatusCode = http.StatusBadRequest

		if c.logger.IsLevelEnabled(logger.DebugLevel) {
			dump, _ := httputil.DumpResponse(&resp, false)
			c.logger.Debug(string(dump))
		}

		resp.Write(c.Conn)
		return errors.New("bad request")
	}

	resp.StatusCode = http.StatusSwitchingProtocols
	resp.Header.Set("Connection", "Upgrade")
	resp.Header.Set("Upgrade", "websocket")
	resp.Header.Set("Sec-WebSocket-Accept", c.computeAcceptKey(r.Header.Get("Sec-WebSocket-Key")))

	if c.logger.IsLevelEnabled(logger.DebugLevel) {
		dump, _ := httputil.DumpResponse(&resp, false)
		c.logger.Debug(string(dump))
	}

	if c.rbuf.Len() > 0 {
		// cache the response header if there are extra data in the request body.
		resp.Write(&c.wbuf)
		return
	}

	err = resp.Write(c.Conn)
	return
}

func (c *obfsHTTPConn) Read(b []byte) (n int, err error) {
	if err = c.Handshake(); err != nil {
		return
	}

	if c.rbuf.Len() > 0 {
		return c.rbuf.Read(b)
	}
	return c.Conn.Read(b)
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

var keyGUID = []byte("258EAFA5-E914-47DA-95CA-C5AB0DC85B11")

func (c *obfsHTTPConn) computeAcceptKey(challengeKey string) string {
	h := sha1.New()
	h.Write([]byte(challengeKey))
	h.Write(keyGUID)
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}
