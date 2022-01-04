package dialer

import (
	"context"
	"crypto/tls"
	"net"
	"net/url"

	"github.com/go-gost/gost/pkg/logger"
)

type Options struct {
	User      *url.Userinfo
	TLSConfig *tls.Config
	Logger    logger.Logger
}

type Option func(opts *Options)

func UserOption(user *url.Userinfo) Option {
	return func(opts *Options) {
		opts.User = user
	}
}

func TLSConfigOption(tlsConfig *tls.Config) Option {
	return func(opts *Options) {
		opts.TLSConfig = tlsConfig
	}
}

func LoggerOption(logger logger.Logger) Option {
	return func(opts *Options) {
		opts.Logger = logger
	}
}

type DialOptions struct {
	DialFunc func(ctx context.Context, addr string) (net.Conn, error)
}

type DialOption func(opts *DialOptions)

func DialFuncDialOption(dialf func(ctx context.Context, addr string) (net.Conn, error)) DialOption {
	return func(opts *DialOptions) {
		opts.DialFunc = dialf
	}
}

type HandshakeOptions struct {
	Addr string
}

type HandshakeOption func(opts *HandshakeOptions)

func AddrHandshakeOption(addr string) HandshakeOption {
	return func(opts *HandshakeOptions) {
		opts.Addr = addr
	}
}
