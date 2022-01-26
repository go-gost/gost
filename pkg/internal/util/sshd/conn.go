package sshd

import (
	"context"
	"errors"
	"net"
	"time"

	"golang.org/x/crypto/ssh"
)

type DirectForwardConn struct {
	conn    ssh.Conn
	channel ssh.Channel
	dstAddr string
}

func NewDirectForwardConn(conn ssh.Conn, channel ssh.Channel, dstAddr string) net.Conn {
	return &DirectForwardConn{
		conn:    conn,
		channel: channel,
		dstAddr: dstAddr,
	}
}

func (c *DirectForwardConn) Read(b []byte) (n int, err error) {
	return c.channel.Read(b)
}

func (c *DirectForwardConn) Write(b []byte) (n int, err error) {
	return c.channel.Write(b)
}

func (c *DirectForwardConn) Close() error {
	return c.channel.Close()
}

func (c *DirectForwardConn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *DirectForwardConn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *DirectForwardConn) SetDeadline(t time.Time) error {
	return &net.OpError{Op: "set", Net: "nop", Source: nil, Addr: nil, Err: errors.New("deadline not supported")}
}

func (c *DirectForwardConn) SetReadDeadline(t time.Time) error {
	return &net.OpError{Op: "set", Net: "nop", Source: nil, Addr: nil, Err: errors.New("deadline not supported")}
}

func (c *DirectForwardConn) SetWriteDeadline(t time.Time) error {
	return &net.OpError{Op: "set", Net: "nop", Source: nil, Addr: nil, Err: errors.New("deadline not supported")}
}

func (c *DirectForwardConn) DstAddr() string {
	return c.dstAddr
}

type RemoteForwardConn struct {
	ctx  context.Context
	conn ssh.Conn
	req  *ssh.Request
}

func NewRemoteForwardConn(ctx context.Context, conn ssh.Conn, req *ssh.Request) net.Conn {
	return &RemoteForwardConn{
		ctx:  ctx,
		conn: conn,
		req:  req,
	}
}

func (c *RemoteForwardConn) Conn() ssh.Conn {
	return c.conn
}

func (c *RemoteForwardConn) Request() *ssh.Request {
	return c.req
}

func (c *RemoteForwardConn) Read(b []byte) (n int, err error) {
	return 0, &net.OpError{Op: "read", Net: "nop", Source: nil, Addr: nil, Err: errors.New("read not supported")}
}

func (c *RemoteForwardConn) Write(b []byte) (n int, err error) {
	return 0, &net.OpError{Op: "write", Net: "nop", Source: nil, Addr: nil, Err: errors.New("write not supported")}
}

func (c *RemoteForwardConn) Close() error {
	return &net.OpError{Op: "close", Net: "nop", Source: nil, Addr: nil, Err: errors.New("close not supported")}
}

func (c *RemoteForwardConn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *RemoteForwardConn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *RemoteForwardConn) SetDeadline(t time.Time) error {
	return &net.OpError{Op: "set", Net: "nop", Source: nil, Addr: nil, Err: errors.New("deadline not supported")}
}

func (c *RemoteForwardConn) SetReadDeadline(t time.Time) error {
	return &net.OpError{Op: "set", Net: "nop", Source: nil, Addr: nil, Err: errors.New("deadline not supported")}
}

func (c *RemoteForwardConn) SetWriteDeadline(t time.Time) error {
	return &net.OpError{Op: "set", Net: "nop", Source: nil, Addr: nil, Err: errors.New("deadline not supported")}
}

func (c *RemoteForwardConn) Done() <-chan struct{} {
	return c.ctx.Done()
}
