package connector

import (
	"net/url"
	"time"

	"github.com/go-gost/gost/pkg/logger"
)

type Options struct {
	User   *url.Userinfo
	Logger logger.Logger
}

type Option func(opts *Options)

func UserOption(user *url.Userinfo) Option {
	return func(opts *Options) {
		opts.User = user
	}
}

func LoggerOption(logger logger.Logger) Option {
	return func(opts *Options) {
		opts.Logger = logger
	}
}

type ConnectOptions struct {
}

type ConnectOption func(opts *ConnectOptions)

type BindOptions struct {
	Mux               bool
	Backlog           int
	UDPDataQueueSize  int
	UDPDataBufferSize int
	UDPConnTTL        time.Duration
}

type BindOption func(opts *BindOptions)

func MuxBindOption(mux bool) BindOption {
	return func(opts *BindOptions) {
		opts.Mux = mux
	}
}

func BacklogBindOption(backlog int) BindOption {
	return func(opts *BindOptions) {
		opts.Backlog = backlog
	}
}

func UDPDataQueueSizeBindOption(size int) BindOption {
	return func(opts *BindOptions) {
		opts.UDPDataQueueSize = size
	}
}

func UDPDataBufferSizeBindOption(size int) BindOption {
	return func(opts *BindOptions) {
		opts.UDPDataBufferSize = size
	}
}

func UDPConnTTLBindOption(ttl time.Duration) BindOption {
	return func(opts *BindOptions) {
		opts.UDPConnTTL = ttl
	}
}
