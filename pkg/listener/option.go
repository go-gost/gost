package listener

import (
	"github.com/go-gost/gost/pkg/logger"
)

type Options struct {
	Addr   string
	Logger logger.Logger
}

type Option func(opts *Options)

func AddrOption(addr string) Option {
	return func(opts *Options) {
		opts.Addr = addr
	}
}

func LoggerOption(logger logger.Logger) Option {
	return func(opts *Options) {
		opts.Logger = logger
	}
}
