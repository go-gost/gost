package tun

import (
	"errors"
	"net"
	"syscall"

	"github.com/docker/libcontainer/netlink"
	"github.com/milosgajdos/tenus"
	"github.com/songgao/water"
)

func (l *tunListener) createTun() (conn net.Conn, itf *net.Interface, err error) {
	ip, ipNet, err := net.ParseCIDR(l.md.net)
	if err != nil {
		return
	}

	ifce, err := water.New(water.Config{
		DeviceType: water.TUN,
		PlatformSpecificParams: water.PlatformSpecificParams{
			Name: l.md.name,
		},
	})
	if err != nil {
		return
	}

	link, err := tenus.NewLinkFrom(ifce.Name())
	if err != nil {
		return
	}

	l.logger.Debugf("ip link set dev %s mtu %d", ifce.Name(), l.md.mtu)

	if err = link.SetLinkMTU(l.md.mtu); err != nil {
		return
	}

	l.logger.Debugf("ip address add %s dev %s", l.md.net, ifce.Name())

	if err = link.SetLinkIp(ip, ipNet); err != nil {
		return
	}

	l.logger.Debugf("ip link set dev %s up", ifce.Name())
	if err = link.SetLinkUp(); err != nil {
		return
	}

	if err = l.addRoutes(ifce.Name(), l.md.routes...); err != nil {
		return
	}

	itf, err = net.InterfaceByName(ifce.Name())
	if err != nil {
		return
	}

	conn = &tunConn{
		ifce: ifce,
		addr: &net.IPAddr{IP: ip},
	}
	return
}

func (l *tunListener) addRoutes(ifName string, routes ...ipRoute) error {
	for _, route := range routes {
		l.logger.Debugf("ip route add %s dev %s", route.Dest.String(), ifName)
		if err := netlink.AddRoute(route.Dest.String(), "", "", ifName); err != nil && !errors.Is(err, syscall.EEXIST) {
			return err
		}
	}
	return nil
}
