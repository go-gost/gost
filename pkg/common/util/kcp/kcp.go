package kcp

import (
	"net"

	"github.com/golang/snappy"
)

type kcpCompStreamConn struct {
	net.Conn
	w *snappy.Writer
	r *snappy.Reader
}

func KCPCompStreamConn(conn net.Conn) net.Conn {
	return &kcpCompStreamConn{
		Conn: conn,
		w:    snappy.NewBufferedWriter(conn),
		r:    snappy.NewReader(conn),
	}
}

func (c *kcpCompStreamConn) Read(b []byte) (n int, err error) {
	return c.r.Read(b)
}

func (c *kcpCompStreamConn) Write(b []byte) (n int, err error) {
	n, err = c.w.Write(b)
	if err != nil {
		return
	}
	err = c.w.Flush()
	return n, err
}
