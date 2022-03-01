package dialer

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/go-gost/gost/pkg/logger"
)

var (
	DefaultNetDialer = &NetDialer{
		Timeout: 30 * time.Second,
	}
)

type NetDialer struct {
	Interface string
	Timeout   time.Duration
	DialFunc  func(ctx context.Context, network, addr string) (net.Conn, error)
}

func (d *NetDialer) Dial(ctx context.Context, network, addr string) (net.Conn, error) {
	ifAddr, err := parseInterfaceAddr(d.Interface, network)
	if err != nil {
		return nil, err
	}
	if d.DialFunc != nil {
		return d.DialFunc(ctx, network, addr)
	}
	logger.Default().Infof("interface: %s %s %v", d.Interface, network, ifAddr)

	switch network {
	case "udp", "udp4", "udp6":
		if addr == "" {
			var laddr *net.UDPAddr
			if ifAddr != nil {
				laddr, _ = ifAddr.(*net.UDPAddr)
			}

			return net.ListenUDP(network, laddr)
		}
	case "tcp", "tcp4", "tcp6":
	default:
		return nil, fmt.Errorf("dial: unsupported network %s", network)
	}
	netd := net.Dialer{
		Timeout:   d.Timeout,
		LocalAddr: ifAddr,
	}
	return netd.DialContext(ctx, network, addr)
}

func parseInterfaceAddr(ifceName, network string) (net.Addr, error) {
	if ifceName == "" {
		return nil, nil
	}

	ip := net.ParseIP(ifceName)
	if ip == nil {
		ifce, err := net.InterfaceByName(ifceName)
		if err != nil {
			return nil, err
		}
		addrs, err := ifce.Addrs()
		if err != nil {
			return nil, err
		}
		if len(addrs) == 0 {
			return nil, fmt.Errorf("addr not found for interface %s", ifceName)
		}
		ip = addrs[0].(*net.IPNet).IP
	}

	switch network {
	case "tcp", "tcp4", "tcp6":
		return &net.TCPAddr{IP: ip}, nil
	case "udp", "udp4", "udp6":
		return &net.UDPAddr{IP: ip}, nil
	default:
		return &net.IPAddr{IP: ip}, nil
	}
}
