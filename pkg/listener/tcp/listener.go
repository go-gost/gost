package tcp

import (
	"net"

	"github.com/go-gost/gost/pkg/internal/utils"
	"github.com/go-gost/gost/pkg/listener"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
)

func init() {
	registry.RegisterListener("tcp", NewListener)
}

type tcpListener struct {
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
	return &tcpListener{
		addr:   options.Addr,
		logger: options.Logger,
	}
}

func (l *tcpListener) Init(md md.Metadata) (err error) {
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

	if l.md.keepAlive {
		l.Listener = &utils.TCPKeepAliveListener{
			TCPListener:     ln,
			KeepAlivePeriod: l.md.keepAlivePeriod,
		}
		return
	}

	l.Listener = ln
	return
}

func (l *tcpListener) parseMetadata(md md.Metadata) (err error) {
	l.md.keepAlive = md.GetBool(keepAlive)
	l.md.keepAlivePeriod = md.GetDuration(keepAlivePeriod)

	return
}
