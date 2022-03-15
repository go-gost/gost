package tcp

import (
	"net"

	"github.com/go-gost/gost/v3/pkg/listener"
	"github.com/go-gost/gost/v3/pkg/logger"
	md "github.com/go-gost/gost/v3/pkg/metadata"
	"github.com/go-gost/gost/v3/pkg/registry"
	metrics "github.com/go-gost/metrics/wrapper"
)

func init() {
	registry.ListenerRegistry().Register("tcp", NewListener)
}

type tcpListener struct {
	ln      net.Listener
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

	l.ln = metrics.WrapListener(l.options.Service, ln)

	return
}

func (l *tcpListener) Accept() (conn net.Conn, err error) {
	return l.ln.Accept()
}

func (l *tcpListener) Addr() net.Addr {
	return l.ln.Addr()
}

func (l *tcpListener) Close() error {
	return l.ln.Close()
}
