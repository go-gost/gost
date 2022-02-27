package http

import (
	"net"

	"github.com/go-gost/gost/pkg/listener"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
)

func init() {
	registry.ListenerRegistry().Register("ohttp", NewListener)
}

type obfsListener struct {
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
	return &obfsListener{
		addr:   options.Addr,
		logger: options.Logger,
	}
}

func (l *obfsListener) Init(md md.Metadata) (err error) {
	if err = l.parseMetadata(md); err != nil {
		return
	}

	laddr, err := net.ResolveTCPAddr("tcp", l.addr)
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
