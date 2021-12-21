package dns

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"time"
)

type Server interface {
	ListenAndServe() error
	Shutdown() error
}

type dohServer struct {
	addr      string
	tlsConfig *tls.Config
	server    *http.Server
}

func (s *dohServer) ListenAndServe() error {
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	ln = tls.NewListener(ln, s.tlsConfig)
	return s.server.Serve(ln)
}

func (s *dohServer) Shutdown() error {
	return s.server.Shutdown(context.Background())
}

type ResponseWriter interface {
	io.Writer
	RemoteAddr() net.Addr
}

type dohResponseWriter struct {
	raddr net.Addr
	http.ResponseWriter
}

func (w *dohResponseWriter) RemoteAddr() net.Addr {
	return w.raddr
}

type serverConn struct {
	r      io.Reader
	w      ResponseWriter
	laddr  net.Addr
	closed chan struct{}
}

func (c *serverConn) Read(b []byte) (n int, err error) {
	select {
	case <-c.closed:
		err = io.ErrClosedPipe
		return
	}
	return c.r.Read(b)
}

func (c *serverConn) Write(b []byte) (n int, err error) {
	select {
	case <-c.closed:
		err = io.ErrClosedPipe
		return
	}
	return c.w.Write(b)
}

func (c *serverConn) Close() error {
	select {
	case <-c.closed:
	default:
		close(c.closed)
	}
	return nil
}

func (c *serverConn) Wait() error {
	<-c.closed
	return nil
}

func (c *serverConn) LocalAddr() net.Addr {
	return c.laddr
}

func (c *serverConn) RemoteAddr() net.Addr {
	return c.w.RemoteAddr()
}

func (c *serverConn) SetDeadline(t time.Time) error {
	return &net.OpError{Op: "set", Net: "dns", Source: nil, Addr: nil, Err: errors.New("deadline not supported")}
}

func (c *serverConn) SetReadDeadline(t time.Time) error {
	return &net.OpError{Op: "set", Net: "dns", Source: nil, Addr: nil, Err: errors.New("deadline not supported")}
}

func (c *serverConn) SetWriteDeadline(t time.Time) error {
	return &net.OpError{Op: "set", Net: "dns", Source: nil, Addr: nil, Err: errors.New("deadline not supported")}
}
