package listener

import (
	"crypto/tls"
	"net/url"

	"github.com/go-gost/gost/pkg/auth"
	"github.com/go-gost/gost/pkg/chain"
	"github.com/go-gost/gost/pkg/logger"
)

type Options struct {
	Addr      string
	Auther    auth.Authenticator
	Auth      *url.Userinfo
	TLSConfig *tls.Config
	Chain     chain.Chainer
	Logger    logger.Logger
}

type Option func(opts *Options)

func AddrOption(addr string) Option {
	return func(opts *Options) {
		opts.Addr = addr
	}
}

func AutherOption(auther auth.Authenticator) Option {
	return func(opts *Options) {
		opts.Auther = auther
	}
}

func AuthOption(auth *url.Userinfo) Option {
	return func(opts *Options) {
		opts.Auth = auth
	}
}

func TLSConfigOption(tlsConfig *tls.Config) Option {
	return func(opts *Options) {
		opts.TLSConfig = tlsConfig
	}
}

func ChainOption(chain chain.Chainer) Option {
	return func(opts *Options) {
		opts.Chain = chain
	}
}

func LoggerOption(logger logger.Logger) Option {
	return func(opts *Options) {
		opts.Logger = logger
	}
}
