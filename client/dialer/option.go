package dialer

import (
	"github.com/go-gost/gost/logger"
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
