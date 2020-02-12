package gost

import (
	"context"
	"net"
	"time"
)

// Client is a proxy client.
// A client is divided into two layers: Connector and Dialer.
// Connector is responsible for connecting to the destination address through this proxy.
// Dialer performs a handshake with this proxy.
type Client struct {
	Connector
	Dialer
}

// Connector is responsible for connecting to the destination address.
type Connector interface {
	Connect(ctx context.Context, conn net.Conn, network, address string, options ...ConnectOption) (net.Conn, error)
}

// Dialer is responsible for dialing and handshaking with the proxy server.
type Dialer interface {
	Dial(ctx context.Context, addr string, options ...DialOption) (net.Conn, error)
	Handshake(ctx context.Context, conn net.Conn, options ...HandshakeOption) (net.Conn, error)
	// Multiplex reports whether the Transporter is multiplex.
	Multiplex() bool
}

// DialOptions describes the options for Transporter.Dial.
type DialOptions struct {
	Timeout time.Duration
	Chain   *Chain
}

// DialOption allows a common way to set DialOptions.
type DialOption func(opts *DialOptions)

// TimeoutDialOption specifies the timeout used by Transporter.Dial
func TimeoutDialOption(timeout time.Duration) DialOption {
	return func(opts *DialOptions) {
		opts.Timeout = timeout
	}
}

// ChainDialOption specifies a chain used by Transporter.Dial
func ChainDialOption(chain *Chain) DialOption {
	return func(opts *DialOptions) {
		opts.Chain = chain
	}
}

// HandshakeOptions describes the options for handshake.
type HandshakeOptions struct {
	Addr     string
	Host     string
	Retry    int
	Timeout  time.Duration
	Interval time.Duration
}

// HandshakeOption allows a common way to set HandshakeOptions.
type HandshakeOption func(opts *HandshakeOptions)

// AddrHandshakeOption specifies the server address
func AddrHandshakeOption(addr string) HandshakeOption {
	return func(opts *HandshakeOptions) {
		opts.Addr = addr
	}
}

// HostHandshakeOption specifies the hostname
func HostHandshakeOption(host string) HandshakeOption {
	return func(opts *HandshakeOptions) {
		opts.Host = host
	}
}

// TimeoutHandshakeOption specifies the timeout used by Transporter.Handshake
func TimeoutHandshakeOption(timeout time.Duration) HandshakeOption {
	return func(opts *HandshakeOptions) {
		opts.Timeout = timeout
	}
}

// IntervalHandshakeOption specifies the interval time used by Transporter.Handshake
func IntervalHandshakeOption(interval time.Duration) HandshakeOption {
	return func(opts *HandshakeOptions) {
		opts.Interval = interval
	}
}

// RetryHandshakeOption specifies the times of retry used by Transporter.Handshake
func RetryHandshakeOption(retry int) HandshakeOption {
	return func(opts *HandshakeOptions) {
		opts.Retry = retry
	}
}

// ConnectOptions describes the options for Connector.Connect.
type ConnectOptions struct {
	Addr    string
	Timeout time.Duration
}

// ConnectOption allows a common way to set ConnectOptions.
type ConnectOption func(opts *ConnectOptions)

// AddrConnectOption specifies the corresponding address of the target.
func AddrConnectOption(addr string) ConnectOption {
	return func(opts *ConnectOptions) {
		opts.Addr = addr
	}
}

// TimeoutConnectOption specifies the timeout for connecting to target.
func TimeoutConnectOption(timeout time.Duration) ConnectOption {
	return func(opts *ConnectOptions) {
		opts.Timeout = timeout
	}
}
