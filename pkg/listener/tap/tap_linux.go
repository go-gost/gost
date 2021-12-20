package tap

import (
	"net"

	"github.com/docker/libcontainer/netlink"
	"github.com/milosgajdos/tenus"
	"github.com/songgao/water"
)

func (l *tapListener) createTap() (ifce *water.Interface, ip net.IP, err error) {
	var ipNet *net.IPNet
	if l.md.config.Net != "" {
		ip, ipNet, err = net.ParseCIDR(l.md.config.Net)
		if err != nil {
			return
		}
	}

	ifce, err = water.New(water.Config{
		DeviceType: water.TAP,
		PlatformSpecificParams: water.PlatformSpecificParams{
			Name: l.md.config.Name,
		},
	})
	if err != nil {
		return
	}

	link, err := tenus.NewLinkFrom(ifce.Name())
	if err != nil {
		return
	}

	l.logger.Debugf("ip link set dev %s mtu %d", ifce.Name(), l.md.config.MTU)

	if err = link.SetLinkMTU(l.md.config.MTU); err != nil {
		return
	}

	if l.md.config.Net != "" {
		l.logger.Debugf("ip address add %s dev %s", l.md.config.Net, ifce.Name())

		if err = link.SetLinkIp(ip, ipNet); err != nil {
			return
		}
	}

	l.logger.Debugf("ip link set dev %s up", ifce.Name())
	if err = link.SetLinkUp(); err != nil {
		return
	}

	if err = l.addRoutes(ifce.Name(), l.md.config.Gateway, l.md.config.Routes...); err != nil {
		return
	}

	return
}

func (l *tapListener) addRoutes(ifName string, gw string, routes ...string) error {
	for _, route := range routes {
		l.logger.Debugf("ip route add %s via %s dev %s", route, gw, ifName)
		if err := netlink.AddRoute(route, "", gw, ifName); err != nil {
			return err
		}
	}
	return nil
}
