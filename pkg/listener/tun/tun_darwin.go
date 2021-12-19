package tun

import (
	"fmt"
	"net"
	"os/exec"
	"strings"

	"github.com/songgao/water"
)

func (l *tunListener) createTun() (conn net.Conn, itf *net.Interface, err error) {
	ip, _, err := net.ParseCIDR(l.md.net)
	if err != nil {
		return
	}

	ifce, err := water.New(water.Config{
		DeviceType: water.TUN,
	})
	if err != nil {
		return
	}

	peer := l.md.peer
	if peer == "" {
		peer = ip.String()
	}

	cmd := fmt.Sprintf("ifconfig %s inet %s %s mtu %d up",
		ifce.Name(), l.md.net, l.md.peer, l.md.mtu)
	l.logger.Debug(cmd)

	args := strings.Split(cmd, " ")
	if err = exec.Command(args[0], args[1:]...).Run(); err != nil {
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
		cmd := fmt.Sprintf("route add -net %s -interface %s", route.Dest.String(), ifName)
		l.logger.Debug(cmd)
		args := strings.Split(cmd, " ")
		if err := exec.Command(args[0], args[1:]...).Run(); err != nil {
			return err
		}
	}
	return nil
}
