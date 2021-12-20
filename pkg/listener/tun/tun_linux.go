package tun

import (
	"errors"
	"net"
	"syscall"

	"github.com/docker/libcontainer/netlink"
	tun_util "github.com/go-gost/gost/pkg/internal/util/tun"
	"github.com/milosgajdos/tenus"
	"github.com/songgao/water"
)

func (l *tunListener) createTun() (ifce *water.Interface, ip net.IP, err error) {
	ip, ipNet, err := net.ParseCIDR(l.md.config.Net)
	if err != nil {
		return
	}

	ifce, err = water.New(water.Config{
		DeviceType: water.TUN,
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

	l.logger.Debugf("ip address add %s dev %s", l.md.config.Net, ifce.Name())

	if err = link.SetLinkIp(ip, ipNet); err != nil {
		return
	}

	l.logger.Debugf("ip link set dev %s up", ifce.Name())
	if err = link.SetLinkUp(); err != nil {
		return
	}

	if err = l.addRoutes(ifce.Name(), l.md.config.Routes...); err != nil {
		return
	}

	return
}

func (l *tunListener) addRoutes(ifName string, routes ...tun_util.Route) error {
	for _, route := range routes {
		l.logger.Debugf("ip route add %s dev %s", route.Net.String(), ifName)
		if err := netlink.AddRoute(route.Net.String(), "", "", ifName); err != nil && !errors.Is(err, syscall.EEXIST) {
			return err
		}
	}
	return nil
}
