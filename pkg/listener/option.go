package listener

import (
	"github.com/go-gost/gost/pkg/chain"
	"github.com/go-gost/gost/pkg/logger"
)

type Options struct {
	Addr   string
	Chain  *chain.Chain
	Logger logger.Logger
}

type Option func(opts *Options)

func AddrOption(addr string) Option {
	return func(opts *Options) {
		opts.Addr = addr
	}
}

func ChainOption(chain *chain.Chain) Option {
	return func(opts *Options) {
		opts.Chain = chain
	}
}

func LoggerOption(logger logger.Logger) Option {
	return func(opts *Options) {
		opts.Logger = logger
	}
}
