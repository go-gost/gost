package dialer

import (
	"context"
	"net"

	"github.com/go-gost/gost/pkg/logger"
)

type Options struct {
	Logger logger.Logger
}

type Option func(opts *Options)

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
