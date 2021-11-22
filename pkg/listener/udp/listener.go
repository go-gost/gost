package udp

import (
	"net"

	"github.com/go-gost/gost/pkg/common/util/udp"
	"github.com/go-gost/gost/pkg/listener"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
)

func init() {
	registry.RegisterListener("udp", NewListener)
}

type udpListener struct {
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
	return &udpListener{
		addr:   options.Addr,
		logger: options.Logger,
	}
}

func (l *udpListener) Init(md md.Metadata) (err error) {
	if err = l.parseMetadata(md); err != nil {
		return
	}

	laddr, err := net.ResolveUDPAddr("udp", l.addr)
	if err != nil {
		return
	}

	conn, err := net.ListenUDP("udp", laddr)
	if err != nil {
		return
	}

	l.Listener = udp.NewListener(conn, laddr,
		l.md.backlog,
		l.md.readQueueSize, l.md.readBufferSize,
		l.md.ttl,
		l.logger)
	return
}
