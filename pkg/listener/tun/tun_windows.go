package tun

import (
	"fmt"
	"net"
	"os/exec"
	"strings"

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
			ComponentID:   "tap0901",
			InterfaceName: l.md.name,
			Network:       l.md.net,
		},
	})
	if err != nil {
		return
	}

	cmd := fmt.Sprintf("netsh interface ip set address name=%s "+
		"source=static addr=%s mask=%s gateway=none",
		ifce.Name(), ip.String(), ipMask(ipNet.Mask))
	l.logger.Debug(cmd)

	args := strings.Split(cmd, " ")
	if er := exec.Command(args[0], args[1:]...).Run(); er != nil {
		err = fmt.Errorf("%s: %v", cmd, er)
		return
	}

	if err = l.addRoutes(ifce.Name(), l.md.gateway, l.md.routes...); err != nil {
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

func (l *tunListener) addRoutes(ifName string, gw string, routes ...ipRoute) error {
	for _, route := range routes {
		l.deleteRoute(ifName, route.Dest.String())

		cmd := fmt.Sprintf("netsh interface ip add route prefix=%s interface=%s store=active",
			route.Dest.String(), ifName)
		if gw != "" {
			cmd += " nexthop=" + gw
		}
		l.logger.Debug(cmd)
		args := strings.Split(cmd, " ")
		if er := exec.Command(args[0], args[1:]...).Run(); er != nil {
			return fmt.Errorf("%s: %v", cmd, er)
		}
	}
	return nil
}

func (l *tunListener) deleteRoute(ifName string, route string) error {
	cmd := fmt.Sprintf("netsh interface ip delete route prefix=%s interface=%s store=active",
		route, ifName)
	l.logger.Debug(cmd)
	args := strings.Split(cmd, " ")
	return exec.Command(args[0], args[1:]...).Run()
}

func ipMask(mask net.IPMask) string {
	return fmt.Sprintf("%d.%d.%d.%d", mask[0], mask[1], mask[2], mask[3])
}
