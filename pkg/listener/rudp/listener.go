package rudp

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
	registry.RegisterListener("rudp", NewListener)
}

type rudpListener struct {
	addr   string
	laddr  *net.UDPAddr
	chain  *chain.Chain
	ln     net.Listener
	md     metadata
	logger logger.Logger
	closed chan struct{}
}

func NewListener(opts ...listener.Option) listener.Listener {
	options := &listener.Options{}
	for _, opt := range opts {
		opt(options)
	}
	return &rudpListener{
		addr:   options.Addr,
		closed: make(chan struct{}),
		logger: options.Logger,
	}
}

// implements listener.Chainable interface
func (l *rudpListener) Chain(chain *chain.Chain) {
	l.chain = chain
}

func (l *rudpListener) Init(md md.Metadata) (err error) {
	if err = l.parseMetadata(md); err != nil {
		return
	}

	laddr, err := net.ResolveUDPAddr("udp", l.addr)
	if err != nil {
		return
	}

	l.laddr = laddr

	return
}

func (l *rudpListener) Accept() (conn net.Conn, err error) {
	select {
	case <-l.closed:
		return nil, net.ErrClosed
	default:
	}

	if l.ln == nil {
		r := (&chain.Router{}).
			WithChain(l.chain).
			WithRetry(l.md.retryCount).
			WithLogger(l.logger)
		l.ln, err = r.Bind(context.Background(), "udp", l.laddr.String(),
			connector.BacklogBindOption(l.md.backlog),
			connector.UDPConnTTLBindOption(l.md.ttl),
			connector.UDPDataBufferSizeBindOption(l.md.readBufferSize),
			connector.UDPDataQueueSizeBindOption(l.md.readQueueSize),
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

func (l *rudpListener) Addr() net.Addr {
	return l.laddr
}

func (l *rudpListener) Close() error {
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
