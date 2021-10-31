package tcp

import (
	"context"
	"net"

	"github.com/go-gost/gost/pkg/components/dialer"
	md "github.com/go-gost/gost/pkg/components/metadata"
	"github.com/go-gost/gost/pkg/logger"
	"github.com/go-gost/gost/pkg/registry"
)

func init() {
	registry.RegisterDialer("tcp", NewDialer)
}

type tcpDialer struct {
	md     metadata
	logger logger.Logger
}

func NewDialer(opts ...dialer.Option) dialer.Dialer {
	options := &dialer.Options{}
	for _, opt := range opts {
		opt(options)
	}

	return &tcpDialer{
		logger: options.Logger,
	}
}

func (d *tcpDialer) Init(md md.Metadata) (err error) {
	return d.parseMetadata(md)
}

func (d *tcpDialer) Dial(ctx context.Context, addr string, opts ...dialer.DialOption) (net.Conn, error) {
	var options dialer.DialOptions
	for _, opt := range opts {
		opt(&options)
	}

	dial := options.DialFunc
	if dial != nil {
		conn, err := dial(ctx, addr)
		if err != nil {
			d.logger.Error(err)
		} else {
			if d.logger.IsLevelEnabled(logger.DebugLevel) {
				d.logger.WithFields(map[string]interface{}{
					"src": conn.LocalAddr().String(),
					"dst": addr,
				}).Debug("dial with dial func")
			}
		}
		return conn, err
	}

	var netd net.Dialer
	conn, err := netd.DialContext(ctx, "tcp", addr)
	if err != nil {
		d.logger.Error(err)
	} else {
		if d.logger.IsLevelEnabled(logger.DebugLevel) {
			d.logger.WithFields(map[string]interface{}{
				"src": conn.LocalAddr().String(),
				"dst": addr,
			}).Debug("dial direct")
		}
	}
	return conn, err
}

func (d *tcpDialer) parseMetadata(md md.Metadata) (err error) {
	return
}
