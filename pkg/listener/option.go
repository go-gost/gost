package listener

import (
	"crypto/tls"
	"net/url"

	"github.com/go-gost/gost/v3/pkg/admission"
	"github.com/go-gost/gost/v3/pkg/auth"
	"github.com/go-gost/gost/v3/pkg/chain"
	"github.com/go-gost/gost/v3/pkg/logger"
)

type Options struct {
	Addr      string
	Auther    auth.Authenticator
	Auth      *url.Userinfo
	TLSConfig *tls.Config
	Admission admission.Admission
	Chain     chain.Chainer
	Logger    logger.Logger
	Service   string
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

func AdmissionOption(admission admission.Admission) Option {
	return func(opts *Options) {
		opts.Admission = admission
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

func ServiceOption(service string) Option {
	return func(opts *Options) {
		opts.Service = service
	}
}
