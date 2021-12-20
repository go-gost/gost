package tun

import (
	"errors"
	"net"
	"time"

	"github.com/songgao/water"
)

type Conn struct {
	config *Config
	ifce   *water.Interface
	laddr  net.Addr
	raddr  net.Addr
}

func NewConn(config *Config, ifce *water.Interface, laddr, raddr net.Addr) *Conn {
	return &Conn{
		config: config,
		ifce:   ifce,
		laddr:  laddr,
		raddr:  raddr,
	}
}

func (c *Conn) Config() *Config {
	return c.config
}

func (c *Conn) Read(b []byte) (n int, err error) {
	return c.ifce.Read(b)
}

func (c *Conn) Write(b []byte) (n int, err error) {
	return c.ifce.Write(b)
}

func (c *Conn) Close() (err error) {
	return c.ifce.Close()
}

func (c *Conn) LocalAddr() net.Addr {
	return c.laddr
}

func (c *Conn) RemoteAddr() net.Addr {
	return c.raddr
}

func (c *Conn) SetDeadline(t time.Time) error {
	return &net.OpError{Op: "set", Net: "tuntap", Source: nil, Addr: nil, Err: errors.New("deadline not supported")}
}

func (c *Conn) SetReadDeadline(t time.Time) error {
	return &net.OpError{Op: "set", Net: "tuntap", Source: nil, Addr: nil, Err: errors.New("deadline not supported")}
}

func (c *Conn) SetWriteDeadline(t time.Time) error {
	return &net.OpError{Op: "set", Net: "tuntap", Source: nil, Addr: nil, Err: errors.New("deadline not supported")}
}
