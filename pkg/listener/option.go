package listener

import (
	"crypto/tls"
	"net/url"

	"github.com/go-gost/gost/pkg/chain"
	"github.com/go-gost/gost/pkg/logger"
)

type Options struct {
	Addr      string
	Auths     []*url.Userinfo
	TLSConfig *tls.Config
	Chain     *chain.Chain
	Logger    logger.Logger
}

type Option func(opts *Options)

func AddrOption(addr string) Option {
	return func(opts *Options) {
		opts.Addr = addr
	}
}

func AuthsOption(auths ...*url.Userinfo) Option {
	return func(opts *Options) {
		opts.Auths = auths
	}
}

func TLSConfigOption(tlsConfig *tls.Config) Option {
	return func(opts *Options) {
		opts.TLSConfig = tlsConfig
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
