package http

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"
)

type conn struct {
	net.Conn
	rbuf           bytes.Buffer
	wbuf           bytes.Buffer
	handshaked     bool
	handshakeMutex sync.Mutex
}

func (c *conn) Handshake() (err error) {
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

func (c *conn) handshake() (err error) {
	br := bufio.NewReader(c.Conn)
	r, err := http.ReadRequest(br)
	if err != nil {
		return
	}
	/*
		if Debug {
			dump, _ := httputil.DumpRequest(r, false)
			log.Logf("[ohttp] %s -> %s\n%s", c.RemoteAddr(), c.LocalAddr(), string(dump))
		}
	*/

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
		// log.Logf("[ohttp] %s -> %s : %v", c.Conn.RemoteAddr(), c.Conn.LocalAddr(), err)
		return
	}

	b := bytes.Buffer{}

	if r.Method != http.MethodGet || r.Header.Get("Upgrade") != "websocket" {
		b.WriteString("HTTP/1.1 503 Service Unavailable\r\n")
		b.WriteString("Content-Length: 0\r\n")
		b.WriteString("Date: " + time.Now().Format(time.RFC1123) + "\r\n")
		b.WriteString("\r\n")

		/*
			if Debug {
				log.Logf("[ohttp] %s <- %s\n%s", c.RemoteAddr(), c.LocalAddr(), b.String())
			}
		*/

		b.WriteTo(c.Conn)
		return errors.New("bad request")
	}

	b.WriteString("HTTP/1.1 101 Switching Protocols\r\n")
	b.WriteString("Server: nginx/1.10.0\r\n")
	b.WriteString("Date: " + time.Now().Format(time.RFC1123) + "\r\n")
	b.WriteString("Connection: Upgrade\r\n")
	b.WriteString("Upgrade: websocket\r\n")
	b.WriteString(fmt.Sprintf("Sec-WebSocket-Accept: %s\r\n", computeAcceptKey(r.Header.Get("Sec-WebSocket-Key"))))
	b.WriteString("\r\n")

	/*
		if Debug {
			log.Logf("[ohttp] %s <- %s\n%s", c.RemoteAddr(), c.LocalAddr(), b.String())
		}
	*/

	if c.rbuf.Len() > 0 {
		c.wbuf = b // cache the response header if there are extra data in the request body.
		return
	}

	_, err = b.WriteTo(c.Conn)
	return
}

func (c *conn) Read(b []byte) (n int, err error) {
	if err = c.Handshake(); err != nil {
		return
	}

	if c.rbuf.Len() > 0 {
		return c.rbuf.Read(b)
	}
	return c.Conn.Read(b)
}

func (c *conn) Write(b []byte) (n int, err error) {
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

func computeAcceptKey(challengeKey string) string {
	h := sha1.New()
	h.Write([]byte(challengeKey))
	h.Write(keyGUID)
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}
