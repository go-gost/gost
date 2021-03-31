package tcp

import (
	"context"
	"net"

	"github.com/go-gost/gost/client/dialer"
	"github.com/go-gost/gost/logger"
)

type Dialer struct {
	md     metadata
	logger logger.Logger
}

func NewDialer(opts ...dialer.Option) *Dialer {
	options := &dialer.Options{}
	for _, opt := range opts {
		opt(options)
	}

	return &Dialer{
		logger: options.Logger,
	}
}

func (d *Dialer) Init(md dialer.Metadata) (err error) {
	d.md, err = d.parseMetadata(md)
	if err != nil {
		return
	}
	return nil
}

func (d *Dialer) Dial(ctx context.Context, addr string) (net.Conn, error) {
	return nil, nil
}

func (d *Dialer) parseMetadata(md dialer.Metadata) (m metadata, err error) {
	return
}
