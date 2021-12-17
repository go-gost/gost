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
	registry.RegisterListener("tls", NewListener)
}

type tlsListener struct {
	addr string
	net.Listener
	logger logger.Logger
	md     metadata
}

func NewListener(opts ...listener.Option) listener.Listener {
	options := &listener.Options{}
	for _, opt := range opts {
		opt(options)
	}
	return &tlsListener{
		addr:   options.Addr,
		logger: options.Logger,
	}
}

func (l *tlsListener) Init(md md.Metadata) (err error) {
	if err = l.parseMetadata(md); err != nil {
		return
	}

	ln, err := net.Listen("tcp", l.addr)
	if err != nil {
		return
	}

	l.Listener = tls.NewListener(ln, l.md.tlsConfig)

	return
}
