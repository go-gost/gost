package tls

import (
	"crypto/tls"
	"net"

	"github.com/go-gost/gost/pkg/listener"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
)

func init() {
	registry.ListenerRegistry().Register("tls", NewListener)
}

type tlsListener struct {
	net.Listener
	logger  logger.Logger
	md      metadata
	options listener.Options
}

func NewListener(opts ...listener.Option) listener.Listener {
	options := listener.Options{}
	for _, opt := range opts {
		opt(&options)
	}
	return &tlsListener{
		logger:  options.Logger,
		options: options,
	}
}

func (l *tlsListener) Init(md md.Metadata) (err error) {
	if err = l.parseMetadata(md); err != nil {
		return
	}

	ln, err := net.Listen("tcp", l.options.Addr)
	if err != nil {
		return
	}

	l.Listener = tls.NewListener(ln, l.options.TLSConfig)

	return
}
