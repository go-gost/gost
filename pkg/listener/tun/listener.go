package tun

import (
	"net"

	tun_util "github.com/go-gost/gost/pkg/internal/util/tun"
	"github.com/go-gost/gost/pkg/listener"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
)

func init() {
	registry.RegisterListener("tun", NewListener)
}

type tunListener struct {
	addr    net.Addr
	cqueue  chan net.Conn
	closed  chan struct{}
	logger  logger.Logger
	md      metadata
	options listener.Options
}

func NewListener(opts ...listener.Option) listener.Listener {
	options := listener.Options{}
	for _, opt := range opts {
		opt(&options)
	}
	return &tunListener{
		logger:  options.Logger,
		options: options,
	}
}

func (l *tunListener) Init(md md.Metadata) (err error) {
	if err = l.parseMetadata(md); err != nil {
		return
	}

	l.addr, err = net.ResolveUDPAddr("udp", l.options.Addr)
	if err != nil {
		return
	}

	ifce, ip, err := l.createTun()
	if err != nil {
		if ifce != nil {
			ifce.Close()
		}
		return
	}

	itf, err := net.InterfaceByName(ifce.Name())
	if err != nil {
		return
	}

	addrs, _ := itf.Addrs()
	l.logger.Infof("name: %s, net: %s, mtu: %d, addrs: %s",
		itf.Name, ip, itf.MTU, addrs)

	l.cqueue = make(chan net.Conn, 1)
	l.closed = make(chan struct{})

	conn := tun_util.NewConn(l.md.config, ifce, l.addr, &net.IPAddr{IP: ip})

	l.cqueue <- conn

	return
}

func (l *tunListener) Accept() (net.Conn, error) {
	select {
	case conn := <-l.cqueue:
		return conn, nil
	case <-l.closed:
	}

	return nil, listener.ErrClosed
}

func (l *tunListener) Addr() net.Addr {
	return l.addr
}

func (l *tunListener) Close() error {
	select {
	case <-l.closed:
		return net.ErrClosed
	default:
		close(l.closed)
	}
	return nil
}
