package handler

import (
	"github.com/go-gost/gost/pkg/bypass"
	"github.com/go-gost/gost/pkg/chain"
	"github.com/go-gost/gost/pkg/logger"
	"github.com/go-gost/gost/pkg/resolver"
)

type Options struct {
	Router   *chain.Router
	Bypass   bypass.Bypass
	Resolver resolver.Resolver
	Logger   logger.Logger
}

type Option func(opts *Options)

func RouterOption(router *chain.Router) Option {
	return func(opts *Options) {
		opts.Router = router
	}
}

func BypassOption(bypass bypass.Bypass) Option {
	return func(opts *Options) {
		opts.Bypass = bypass
	}
}

func LoggerOption(logger logger.Logger) Option {
	return func(opts *Options) {
		opts.Logger = logger
	}
}
