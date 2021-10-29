package tls

import (
	"crypto/tls"
	"net"

	"github.com/go-gost/gost/pkg/components/internal/utils"
	"github.com/go-gost/gost/pkg/components/listener"
	md "github.com/go-gost/gost/pkg/components/metadata"
	"github.com/go-gost/gost/pkg/logger"
	"github.com/go-gost/gost/pkg/registry"
)

func init() {
	registry.RegisterListener("tls", NewListener)
}

type Listener struct {
	addr string
	md   metadata
	net.Listener
	logger logger.Logger
}

func NewListener(opts ...listener.Option) listener.Listener {
	options := &listener.Options{}
	for _, opt := range opts {
		opt(options)
	}
	return &Listener{
		addr:   options.Addr,
		logger: options.Logger,
	}
}

func (l *Listener) Init(md md.Metadata) (err error) {
	if err = l.parseMetadata(md); err != nil {
		return
	}

	ln, err := net.Listen("tcp", l.addr)
	if err != nil {
		return
	}

	ln = tls.NewListener(
		&utils.TCPKeepAliveListener{
			TCPListener:     ln.(*net.TCPListener),
			KeepAlivePeriod: l.md.keepAlivePeriod,
		},
		l.md.tlsConfig,
	)

	l.Listener = ln
	return
}

func (l *Listener) parseMetadata(md md.Metadata) (err error) {
	l.md.tlsConfig, err = utils.LoadTLSConfig(
		md.GetString(certFile),
		md.GetString(keyFile),
		md.GetString(caFile),
	)
	if err != nil {
		return
	}

	l.md.keepAlivePeriod = md.GetDuration(keepAlivePeriod)
	return
}
