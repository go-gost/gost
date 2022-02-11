package handler

import (
	"crypto/tls"
	"net/url"

	"github.com/go-gost/gost/pkg/auth"
	"github.com/go-gost/gost/pkg/bypass"
	"github.com/go-gost/gost/pkg/chain"
	"github.com/go-gost/gost/pkg/hosts"
	"github.com/go-gost/gost/pkg/logger"
	"github.com/go-gost/gost/pkg/resolver"
)

type Options struct {
	Retries   int
	Chain     chain.Chainer
	Resolver  resolver.Resolver
	Hosts     hosts.HostMapper
	Bypass    bypass.Bypass
	Auth      *url.Userinfo
	Auther    auth.Authenticator
	TLSConfig *tls.Config
	Logger    logger.Logger
}

type Option func(opts *Options)

func RetriesOption(retries int) Option {
	return func(opts *Options) {
		opts.Retries = retries
	}
}

func ChainOption(chain chain.Chainer) Option {
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

func AuthOption(auth *url.Userinfo) Option {
	return func(opts *Options) {
		opts.Auth = auth
	}
}
func AutherOption(auther auth.Authenticator) Option {
	return func(opts *Options) {
		opts.Auther = auther
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
