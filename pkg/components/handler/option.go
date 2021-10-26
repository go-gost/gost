package handler

import (
	"github.com/go-gost/gost/pkg/chain"
	"github.com/go-gost/gost/pkg/logger"
)

type Options struct {
	Chain  *chain.Chain
	Logger logger.Logger
}

type Option func(opts *Options)

func LoggerOption(logger logger.Logger) Option {
	return func(opts *Options) {
		opts.Logger = logger
	}
}

func ChainOption(chain *chain.Chain) Option {
	return func(opts *Options) {
		opts.Chain = chain
	}
}
