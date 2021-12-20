//go:build !linux && !windows && !darwin

package tap

import (
	"fmt"
	"net"
	"os/exec"
	"strings"

	"github.com/songgao/water"
)

func (l *tapListener) createTap() (ifce *water.Interface, ip net.IP, err error) {
	ip, _, _ = net.ParseCIDR(l.md.config.Net)

	ifce, err = water.New(water.Config{
		DeviceType: water.TAP,
	})
	if err != nil {
		return
	}

	var cmd string
	if l.md.config.Net != "" {
		cmd = fmt.Sprintf("ifconfig %s inet %s mtu %d up", ifce.Name(), l.md.config.Net, l.md.config.MTU)
	} else {
		cmd = fmt.Sprintf("ifconfig %s mtu %d up", ifce.Name(), l.md.config.MTU)
	}
	l.logger.Debug(cmd)

	args := strings.Split(cmd, " ")
	if er := exec.Command(args[0], args[1:]...).Run(); er != nil {
		err = fmt.Errorf("%s: %v", cmd, er)
		return
	}

	if err = l.addRoutes(ifce.Name(), l.md.config.Gateway, l.md.config.Routes...); err != nil {
		return
	}

	return
}

func (l *tapListener) addRoutes(ifName string, gw string, routes ...string) error {
	for _, route := range routes {
		if route == "" {
			continue
		}
		cmd := fmt.Sprintf("route add -net %s dev %s", route, ifName)
		if gw != "" {
			cmd += " gw " + gw
		}
		l.logger.Debug(cmd)
		args := strings.Split(cmd, " ")
		if er := exec.Command(args[0], args[1:]...).Run(); er != nil {
			return fmt.Errorf("%s: %v", cmd, er)
		}
	}
	return nil
}
