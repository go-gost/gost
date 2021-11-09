package udp

import (
	"context"
	"net"

	"github.com/go-gost/gost/pkg/dialer"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
)

func init() {
	registry.RegisterDialer("udp", NewDialer)
}

type udpDialer struct {
	md     metadata
	logger logger.Logger
}

func NewDialer(opts ...dialer.Option) dialer.Dialer {
	options := &dialer.Options{}
	for _, opt := range opts {
		opt(options)
	}

	return &udpDialer{
		logger: options.Logger,
	}
}

func (d *udpDialer) Init(md md.Metadata) (err error) {
	return d.parseMetadata(md)
}

func (d *udpDialer) Dial(ctx context.Context, addr string, opts ...dialer.DialOption) (net.Conn, error) {
	taddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}

	c, err := net.DialUDP("udp", nil, taddr)
	if err != nil {
		return nil, err
	}
	return &conn{
		UDPConn: c,
	}, nil
}

func (d *udpDialer) parseMetadata(md md.Metadata) (err error) {
	return
}
