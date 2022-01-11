package http2

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"
)

// a dummy HTTP2 client conn used by HTTP2 client connector
type ClientConn struct {
	localAddr  net.Addr
	remoteAddr net.Addr
	client     *http.Client
	onClose    func()
}

func NewClientConn(localAddr, remoteAddr net.Addr, client *http.Client, onClose func()) net.Conn {
	return &ClientConn{
		localAddr:  localAddr,
		remoteAddr: remoteAddr,
		client:     client,
		onClose:    onClose,
	}
}

func (c *ClientConn) Client() *http.Client {
	return c.client
}

func (c *ClientConn) Close() error {
	if c.onClose != nil {
		c.onClose()
	}
	return nil
}

func (c *ClientConn) Read(b []byte) (n int, err error) {
	return 0, &net.OpError{Op: "read", Net: "nop", Source: nil, Addr: nil, Err: errors.New("read not supported")}
}

func (c *ClientConn) Write(b []byte) (n int, err error) {
	return 0, &net.OpError{Op: "write", Net: "nop", Source: nil, Addr: nil, Err: errors.New("write not supported")}
}

func (c *ClientConn) LocalAddr() net.Addr {
	return c.localAddr
}

func (c *ClientConn) RemoteAddr() net.Addr {
	return c.remoteAddr
}

func (c *ClientConn) SetDeadline(t time.Time) error {
	return &net.OpError{Op: "set", Net: "nop", Source: nil, Addr: nil, Err: errors.New("deadline not supported")}
}

func (c *ClientConn) SetReadDeadline(t time.Time) error {
	return &net.OpError{Op: "set", Net: "nop", Source: nil, Addr: nil, Err: errors.New("deadline not supported")}
}

func (c *ClientConn) SetWriteDeadline(t time.Time) error {
	return &net.OpError{Op: "set", Net: "nop", Source: nil, Addr: nil, Err: errors.New("deadline not supported")}
}

// a dummy HTTP2 server conn used by HTTP2 handler
type ServerConn struct {
	r          *http.Request
	w          http.ResponseWriter
	localAddr  net.Addr
	remoteAddr net.Addr
	cancel     context.CancelFunc
}

func NewServerConn(w http.ResponseWriter, r *http.Request, localAddr, remoteAddr net.Addr) *ServerConn {
	ctx, cancel := context.WithCancel(r.Context())

	return &ServerConn{
		r:          r.Clone(ctx),
		w:          w,
		localAddr:  localAddr,
		remoteAddr: remoteAddr,
		cancel:     cancel,
	}
}

func (c *ServerConn) Done() <-chan struct{} {
	return c.r.Context().Done()
}

func (c *ServerConn) Request() *http.Request {
	return c.r
}

func (c *ServerConn) Writer() http.ResponseWriter {
	return c.w
}

func (c *ServerConn) Read(b []byte) (n int, err error) {
	return 0, &net.OpError{Op: "read", Net: "http2", Source: nil, Addr: nil, Err: errors.New("read not supported")}
}

func (c *ServerConn) Write(b []byte) (n int, err error) {
	return 0, &net.OpError{Op: "write", Net: "http2", Source: nil, Addr: nil, Err: errors.New("write not supported")}
}

func (c *ServerConn) Close() error {
	c.cancel()

	select {
	case <-c.r.Context().Done():
	default:
	}
	return nil
}

func (c *ServerConn) LocalAddr() net.Addr {
	return c.localAddr
}

func (c *ServerConn) RemoteAddr() net.Addr {
	return c.remoteAddr
}

func (c *ServerConn) SetDeadline(t time.Time) error {
	return &net.OpError{Op: "set", Net: "http2", Source: nil, Addr: nil, Err: errors.New("deadline not supported")}
}

func (c *ServerConn) SetReadDeadline(t time.Time) error {
	return &net.OpError{Op: "set", Net: "http2", Source: nil, Addr: nil, Err: errors.New("deadline not supported")}
}

func (c *ServerConn) SetWriteDeadline(t time.Time) error {
	return &net.OpError{Op: "set", Net: "http2", Source: nil, Addr: nil, Err: errors.New("deadline not supported")}
}
