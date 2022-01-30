package tcp

import (
	"net"

	"github.com/go-gost/gost/pkg/listener"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
)

func init() {
	registry.RegisterListener("tcp", NewListener)
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

	laddr, err := net.ResolveTCPAddr("tcp", l.options.Addr)
	if err != nil {
		return
	}
	ln, err := net.ListenTCP("tcp", laddr)
	if err != nil {
		return
	}

	l.Listener = ln
	return
}
