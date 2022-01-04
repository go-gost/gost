package handler

import (
	"net/url"

	"github.com/go-gost/gost/pkg/bypass"
	"github.com/go-gost/gost/pkg/chain"
	"github.com/go-gost/gost/pkg/hosts"
	"github.com/go-gost/gost/pkg/logger"
	"github.com/go-gost/gost/pkg/resolver"
)

type Options struct {
	Retries  int
	Chain    *chain.Chain
	Resolver resolver.Resolver
	Hosts    hosts.HostMapper
	Bypass   bypass.Bypass
	Auths    []*url.Userinfo
	Logger   logger.Logger
}

type Option func(opts *Options)

func RetriesOption(retries int) Option {
	return func(opts *Options) {
		opts.Retries = retries
	}
}

func ChainOption(chain *chain.Chain) Option {
	return func(opts *Options) {
		opts.Chain = chain
	}
}

func ResolverOption(resolver resolver.Resolver) Option {
	return func(opts *Options) {
		opts.Resolver = resolver
	}
}

func HostsOption(hosts hosts.HostMapper) Option {
	return func(opts *Options) {
		opts.Hosts = hosts
	}
}

func BypassOption(bypass bypass.Bypass) Option {
	return func(opts *Options) {
		opts.Bypass = bypass
	}
}

func AuthsOption(auths ...*url.Userinfo) Option {
	return func(opts *Options) {
		opts.Auths = auths
	}
}

func LoggerOption(logger logger.Logger) Option {
	return func(opts *Options) {
		opts.Logger = logger
	}
}
