package listener

import (
	"net/url"

	"github.com/go-gost/gost/pkg/logger"
)

type Options struct {
	Addr   string
	Auths  []*url.Userinfo
	Logger logger.Logger
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

func LoggerOption(logger logger.Logger) Option {
	return func(opts *Options) {
		opts.Logger = logger
	}
}
