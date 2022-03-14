package dialer

import (
	"context"
	"fmt"
	"net"
	"syscall"
	"time"

	"github.com/go-gost/gost/v3/pkg/logger"
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
	Logger    logger.Logger
}

func (d *NetDialer) Dial(ctx context.Context, network, addr string) (net.Conn, error) {
	if d == nil {
		d = DefaultNetDialer
	}
	log := d.Logger
	if log == nil {
		log = logger.Default()
	}

	ifceName, ifAddr, err := parseInterfaceAddr(d.Interface, network)
	if err != nil {
		return nil, err
	}
	if d.DialFunc != nil {
		return d.DialFunc(ctx, network, addr)
	}
	logger.Default().Infof("interface: %s %v/%s", ifceName, ifAddr, network)

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
		Control: func(network, address string, c syscall.RawConn) error {
			var cerr error
			err := c.Control(func(fd uintptr) {
				cerr = bindDevice(fd, ifceName)
			})
			if err != nil {
				return err
			}
			if cerr != nil {
				return cerr
			}
			return nil
		},
	}
	return netd.DialContext(ctx, network, addr)
}

func parseInterfaceAddr(ifceName, network string) (ifce string, addr net.Addr, err error) {
	if ifceName == "" {
		return
	}

	ip := net.ParseIP(ifceName)
	if ip == nil {
		var ife *net.Interface
		ife, err = net.InterfaceByName(ifceName)
		if err != nil {
			return
		}
		var addrs []net.Addr
		addrs, err = ife.Addrs()
		if err != nil {
			return
		}
		if len(addrs) == 0 {
			err = fmt.Errorf("addr not found for interface %s", ifceName)
			return
		}
		ip = addrs[0].(*net.IPNet).IP
		ifce = ifceName
	} else {
		ifce, err = findInterfaceByIP(ip)
		if err != nil {
			return
		}
	}

	port := 0
	switch network {
	case "tcp", "tcp4", "tcp6":
		addr = &net.TCPAddr{IP: ip, Port: port}
		return
	case "udp", "udp4", "udp6":
		addr = &net.UDPAddr{IP: ip, Port: port}
		return
	default:
		addr = &net.IPAddr{IP: ip}
		return
	}
}

func findInterfaceByIP(ip net.IP) (string, error) {
	ifces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, ifce := range ifces {
		addrs, _ := ifce.Addrs()
		if len(addrs) == 0 {
			continue
		}
		for _, addr := range addrs {
			ipAddr, _ := addr.(*net.IPNet)
			if ipAddr == nil {
				continue
			}
			// logger.Default().Infof("%s-%s", ipAddr, ip)
			if ipAddr.IP.Equal(ip) {
				return ifce.Name, nil
			}
		}
	}
	return "", nil
}
