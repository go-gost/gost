package tls

import (
	"crypto/tls"
	"net"

	"github.com/go-gost/gost/pkg/common/util"
	tls_util "github.com/go-gost/gost/pkg/common/util/tls"
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
	md   metadata
	net.Listener
	logger logger.Logger
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

	ln = tls.NewListener(
		&util.TCPKeepAliveListener{
			TCPListener:     ln.(*net.TCPListener),
			KeepAlivePeriod: l.md.keepAlivePeriod,
		},
		l.md.tlsConfig,
	)

	l.Listener = ln
	return
}

func (l *tlsListener) parseMetadata(md md.Metadata) (err error) {
	l.md.tlsConfig, err = tls_util.LoadTLSConfig(
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
