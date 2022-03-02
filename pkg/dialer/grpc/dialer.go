package grpc

import (
	"context"
	"net"
	"sync"
	"time"

	pb "github.com/go-gost/gost/pkg/common/util/grpc/proto"
	"github.com/go-gost/gost/pkg/dialer"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

func init() {
	registry.DialerRegistry().Register("grpc", NewDialer)
}

type grpcDialer struct {
	clients     map[string]pb.GostTunelClient
	clientMutex sync.Mutex
	md          metadata
	options     dialer.Options
}

func NewDialer(opts ...dialer.Option) dialer.Dialer {
	options := dialer.Options{}
	for _, opt := range opts {
		opt(&options)
	}

	return &grpcDialer{
		clients: make(map[string]pb.GostTunelClient),
		options: options,
	}
}

func (d *grpcDialer) Init(md md.Metadata) (err error) {
	return d.parseMetadata(md)
}

// Multiplex implements dialer.Multiplexer interface.
func (d *grpcDialer) Multiplex() bool {
	return true
}

func (d *grpcDialer) Dial(ctx context.Context, addr string, opts ...dialer.DialOption) (net.Conn, error) {
	remoteAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, err
	}

	d.clientMutex.Lock()
	defer d.clientMutex.Unlock()

	client, ok := d.clients[addr]
	if !ok {
		var options dialer.DialOptions
		for _, opt := range opts {
			opt(&options)
		}

		host := d.md.host
		if host == "" {
			host = options.Host
		}
		if h, _, _ := net.SplitHostPort(host); h != "" {
			host = h
		}

		grpcOpts := []grpc.DialOption{
			// grpc.WithBlock(),
			grpc.WithContextDialer(func(c context.Context, s string) (net.Conn, error) {
				return options.NetDialer.Dial(c, "tcp", s)
			}),
			grpc.WithAuthority(host),
			grpc.WithConnectParams(grpc.ConnectParams{
				Backoff:           backoff.DefaultConfig,
				MinConnectTimeout: 10 * time.Second,
			}),
			grpc.FailOnNonTempDialError(true),
		}
		if !d.md.insecure {
			grpcOpts = append(grpcOpts, grpc.WithTransportCredentials(credentials.NewTLS(d.options.TLSConfig)))
		} else {
			grpcOpts = append(grpcOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
		}

		cc, err := grpc.DialContext(ctx, addr, grpcOpts...)
		if err != nil {
			d.options.Logger.Error(err)
			return nil, err
		}
		client = pb.NewGostTunelClient(cc)
		d.clients[addr] = client
	}

	cli, err := client.Tunnel(ctx)
	if err != nil {
		return nil, err
	}

	return &conn{
		c:          cli,
		localAddr:  &net.TCPAddr{},
		remoteAddr: remoteAddr,
		closed:     make(chan struct{}),
	}, nil
}
