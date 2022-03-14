package tcp

import (
	"net"

	"github.com/go-gost/gost/v3/pkg/common/metrics"
	"github.com/go-gost/gost/v3/pkg/listener"
	"github.com/go-gost/gost/v3/pkg/logger"
	md "github.com/go-gost/gost/v3/pkg/metadata"
	"github.com/go-gost/gost/v3/pkg/registry"
)

func init() {
	registry.ListenerRegistry().Register("tcp", NewListener)
}

type tcpListener struct {
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
	return &tcpListener{
		logger:  options.Logger,
		options: options,
	}
}

func (l *tcpListener) Init(md md.Metadata) (err error) {
	if err = l.parseMetadata(md); err != nil {
		return
	}

	ln, err := net.Listen("tcp", l.options.Addr)
	if err != nil {
		return
	}

	ln = metrics.WrapListener(l.options.Service, ln)
	l.Listener = ln

	return
}
