package http2

import (
	"errors"
	"io"
	"net"
	"time"
)

// HTTP2 connection, wrapped up just like a net.Conn.
type http2Conn struct {
	r          io.Reader
	w          io.Writer
	remoteAddr net.Addr
	localAddr  net.Addr
}

func (c *http2Conn) Read(b []byte) (n int, err error) {
	return c.r.Read(b)
}

func (c *http2Conn) Write(b []byte) (n int, err error) {
	return c.w.Write(b)
}

func (c *http2Conn) Close() (err error) {
	if r, ok := c.r.(io.Closer); ok {
		err = r.Close()
	}
	if w, ok := c.w.(io.Closer); ok {
		err = w.Close()
	}
	return
}

func (c *http2Conn) LocalAddr() net.Addr {
	return c.localAddr
}

func (c *http2Conn) RemoteAddr() net.Addr {
	return c.remoteAddr
}

func (c *http2Conn) SetDeadline(t time.Time) error {
	return &net.OpError{Op: "set", Net: "http2", Source: nil, Addr: nil, Err: errors.New("deadline not supported")}
}

func (c *http2Conn) SetReadDeadline(t time.Time) error {
	return &net.OpError{Op: "set", Net: "http2", Source: nil, Addr: nil, Err: errors.New("deadline not supported")}
}

func (c *http2Conn) SetWriteDeadline(t time.Time) error {
	return &net.OpError{Op: "set", Net: "http2", Source: nil, Addr: nil, Err: errors.New("deadline not supported")}
}
