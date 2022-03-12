package http

import (
	"net"

	"github.com/go-gost/gost/pkg/common/admission"
	"github.com/go-gost/gost/pkg/common/metrics"
	"github.com/go-gost/gost/pkg/listener"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
)

func init() {
	registry.ListenerRegistry().Register("ohttp", NewListener)
}

type obfsListener struct {
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
	return &obfsListener{
		logger:  options.Logger,
		options: options,
	}
}

func (l *obfsListener) Init(md md.Metadata) (err error) {
	if err = l.parseMetadata(md); err != nil {
		return
	}

	ln, err := net.Listen("tcp", l.options.Addr)
	if err != nil {
		return
	}
	ln = metrics.WrapListener(l.options.Service, ln)
	ln = admission.WrapListener(l.options.Admission, ln)

	l.Listener = ln
	return
}

func (l *obfsListener) Accept() (net.Conn, error) {
	c, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}

	return &obfsHTTPConn{
		Conn:   c,
		header: l.md.header,
		logger: l.logger,
	}, nil
}
