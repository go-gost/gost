package http3

import (
	"net"

	pht_util "github.com/go-gost/gost/pkg/internal/util/pht"
	"github.com/go-gost/gost/pkg/listener"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
	"github.com/lucas-clemente/quic-go"
)

func init() {
	registry.RegisterListener("http3", NewListener)
	registry.RegisterListener("h3", NewListener)
}

type http3Listener struct {
	addr    net.Addr
	server  *pht_util.Server
	logger  logger.Logger
	md      metadata
	options listener.Options
}

func NewListener(opts ...listener.Option) listener.Listener {
	options := listener.Options{}
	for _, opt := range opts {
		opt(&options)
	}
	return &http3Listener{
		logger:  options.Logger,
		options: options,
	}
}

func (l *http3Listener) Init(md md.Metadata) (err error) {
	if err = l.parseMetadata(md); err != nil {
		return
	}

	l.addr, err = net.ResolveUDPAddr("udp", l.options.Addr)
	if err != nil {
		return
	}

	l.server = pht_util.NewHTTP3Server(
		l.options.Addr,
		&quic.Config{},
		pht_util.TLSConfigServerOption(l.options.TLSConfig),
		pht_util.BacklogServerOption(l.md.backlog),
		pht_util.PathServerOption(l.md.authorizePath, l.md.pushPath, l.md.pullPath),
		pht_util.LoggerServerOption(l.options.Logger),
	)

	go func() {
		if err := l.server.ListenAndServe(); err != nil {
			l.logger.Error(err)
		}
	}()

	return
}

func (l *http3Listener) Accept() (conn net.Conn, err error) {
	return l.server.Accept()
}

func (l *http3Listener) Addr() net.Addr {
	return l.addr
}

func (l *http3Listener) Close() (err error) {
	return l.server.Close()
}
