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
	registry.ListenerRegistry().Register("rtcp", NewListener)
}

type rtcpListener struct {
	laddr   net.Addr
	ln      net.Listener
	md      metadata
	router  *chain.Router
	logger  logger.Logger
	closed  chan struct{}
	options listener.Options
}

func NewListener(opts ...listener.Option) listener.Listener {
	options := listener.Options{}
	for _, opt := range opts {
		opt(&options)
	}
	return &rtcpListener{
		closed:  make(chan struct{}),
		logger:  options.Logger,
		options: options,
	}
}

func (l *rtcpListener) Init(md md.Metadata) (err error) {
	if err = l.parseMetadata(md); err != nil {
		return
	}

	laddr, err := net.ResolveTCPAddr("tcp", l.options.Addr)
	if err != nil {
		return
	}

	l.laddr = laddr
	l.router = (&chain.Router{}).
		WithChain(l.options.Chain).
		WithLogger(l.logger)

	return
}

func (l *rtcpListener) Accept() (conn net.Conn, err error) {
	select {
	case <-l.closed:
		return nil, net.ErrClosed
	default:
	}

	if l.ln == nil {
		l.ln, err = l.router.Bind(context.Background(), "tcp", l.laddr.String(),
			connector.MuxBindOption(true),
		)
		if err != nil {
			return nil, connector.NewAcceptError(err)
		}
	}
	conn, err = l.ln.Accept()
	if err != nil {
		l.ln.Close()
		l.ln = nil
		return nil, connector.NewAcceptError(err)
	}
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
		if l.ln != nil {
			l.ln.Close()
			l.ln = nil
		}
	}

	return nil
}
