package handler

import (
	"crypto/tls"
	"net/url"

	"github.com/go-gost/gost/v3/pkg/auth"
	"github.com/go-gost/gost/v3/pkg/bypass"
	"github.com/go-gost/gost/v3/pkg/chain"
	"github.com/go-gost/gost/v3/pkg/logger"
	"github.com/go-gost/gost/v3/pkg/metadata"
)

type Options struct {
	Bypass    bypass.Bypass
	Router    *chain.Router
	Auth      *url.Userinfo
	Auther    auth.Authenticator
	TLSConfig *tls.Config
	Logger    logger.Logger
}

type Option func(opts *Options)

func BypassOption(bypass bypass.Bypass) Option {
	return func(opts *Options) {
		opts.Bypass = bypass
	}
}

func RouterOption(router *chain.Router) Option {
	return func(opts *Options) {
		opts.Router = router
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

type HandleOptions struct {
	Metadata metadata.Metadata
}

type HandleOption func(opts *HandleOptions)

func MetadataHandleOption(md metadata.Metadata) HandleOption {
	return func(opts *HandleOptions) {
		opts.Metadata = md
	}
}
