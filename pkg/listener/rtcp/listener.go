package rtcp

import (
	"context"
	"net"

	"github.com/go-gost/gost/pkg/chain"
	"github.com/go-gost/gost/pkg/connector"
	"github.com/go-gost/gost/pkg/listener"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
)

func init() {
	registry.RegisterListener("rtcp", NewListener)
}

type rtcpListener struct {
	addr     string
	laddr    net.Addr
	chain    *chain.Chain
	accepter connector.Accepter
	md       metadata
	logger   logger.Logger
	closed   chan struct{}
}

func NewListener(opts ...listener.Option) listener.Listener {
	options := &listener.Options{}
	for _, opt := range opts {
		opt(options)
	}
	return &rtcpListener{
		addr:   options.Addr,
		closed: make(chan struct{}),
		logger: options.Logger,
	}
}

// implements listener.Chainable interface
func (l *rtcpListener) Chain(chain *chain.Chain) {
	l.chain = chain
}

func (l *rtcpListener) Init(md md.Metadata) (err error) {
	if err = l.parseMetadata(md); err != nil {
		return
	}

	laddr, err := net.ResolveTCPAddr("tcp", l.addr)
	if err != nil {
		return
	}

	l.laddr = laddr

	return
}

func (l *rtcpListener) Addr() net.Addr {
	return l.laddr
}

func (l *rtcpListener) Close() error {
	select {
	case <-l.closed:
	default:
		close(l.closed)
	}

	return nil
}

func (l *rtcpListener) Accept() (conn net.Conn, err error) {
	if l.accepter == nil {
		r := (&chain.Router{}).
			WithChain(l.chain).
			WithRetry(l.md.retryCount).
			WithLogger(l.logger)
		l.accepter, err = r.Bind(context.Background(), "tcp", l.laddr.String())
		if err != nil {
			return nil, connector.NewAcceptError(err)
		}
	}
	conn, err = l.accepter.Accept()
	if err != nil {
		l.accepter.Close()
		l.accepter = nil
		return nil, connector.NewAcceptError(err)
	}
	return
}
