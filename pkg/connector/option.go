package connector

import (
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

type ConnectOptions struct {
}

type ConnectOption func(opts *ConnectOptions)

type BindOptions struct {
	Mux bool
}

type BindOption func(opts *BindOptions)

func MuxBindOption(mux bool) BindOption {
	return func(opts *BindOptions) {
		opts.Mux = mux
	}
}
