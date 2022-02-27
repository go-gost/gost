package ftcp

import (
	"context"
	"net"

	"github.com/go-gost/gost/pkg/dialer"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
	"github.com/xtaci/tcpraw"
)

func init() {
	registry.DialerRegistry().Register("ftcp", NewDialer)
}

type ftcpDialer struct {
	md     metadata
	logger logger.Logger
}

func NewDialer(opts ...dialer.Option) dialer.Dialer {
	options := &dialer.Options{}
	for _, opt := range opts {
		opt(options)
	}

	return &ftcpDialer{
		logger: options.Logger,
	}
}

func (d *ftcpDialer) Init(md md.Metadata) (err error) {
	return d.parseMetadata(md)
}

func (d *ftcpDialer) Dial(ctx context.Context, addr string, opts ...dialer.DialOption) (conn net.Conn, err error) {
	raddr, er := net.ResolveTCPAddr("tcp", addr)
	if er != nil {
		return nil, er
	}
	c, err := tcpraw.Dial("tcp", addr)
	if err != nil {
		return
	}
	return &fakeTCPConn{
		raddr:      raddr,
		PacketConn: c,
	}, nil
}
