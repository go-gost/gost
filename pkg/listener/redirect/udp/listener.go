package udp

import (
	"net"

	"github.com/go-gost/gost/pkg/listener"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
)

func init() {
	registry.RegisterListener("redu", NewListener)
}

type redirectListener struct {
	addr   string
	ln     *net.UDPConn
	logger logger.Logger
	md     metadata
}

func NewListener(opts ...listener.Option) listener.Listener {
	options := &listener.Options{}
	for _, opt := range opts {
		opt(options)
	}
	return &redirectListener{
		addr:   options.Addr,
		logger: options.Logger,
	}
}

func (l *redirectListener) Init(md md.Metadata) (err error) {
	if err = l.parseMetadata(md); err != nil {
		return
	}

	laddr, err := net.ResolveUDPAddr("udp", l.addr)
	if err != nil {
		return
	}

	ln, err := l.listenUDP(laddr)
	if err != nil {
		return
	}

	l.ln = ln
	return
}

func (l *redirectListener) Accept() (conn net.Conn, err error) {
	return l.accept()
}

func (l *redirectListener) Addr() net.Addr {
	return l.ln.LocalAddr()
}

func (l *redirectListener) Close() error {
	return l.ln.Close()
}
