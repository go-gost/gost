package tun

import (
	"net"
	"strings"

	tun_util "github.com/go-gost/gost/pkg/internal/util/tun"
	mdata "github.com/go-gost/gost/pkg/metadata"
)

const (
	DefaultMTU = 1350
)

type metadata struct {
	config *tun_util.Config
}

func (l *tunListener) parseMetadata(md mdata.Metadata) (err error) {
	const (
		name    = "name"
		netKey  = "net"
		peer    = "peer"
		mtu     = "mtu"
		routes  = "routes"
		gateway = "gw"
	)

	config := &tun_util.Config{
		Name:    mdata.GetString(md, name),
		Net:     mdata.GetString(md, netKey),
		Peer:    mdata.GetString(md, peer),
		MTU:     mdata.GetInt(md, mtu),
		Gateway: mdata.GetString(md, gateway),
	}
	if config.MTU <= 0 {
		config.MTU = DefaultMTU
	}

	gw := net.ParseIP(config.Gateway)

	for _, s := range mdata.GetStrings(md, routes) {
		ss := strings.SplitN(s, " ", 2)
		if len(ss) == 2 {
			var route tun_util.Route
			_, ipNet, _ := net.ParseCIDR(strings.TrimSpace(ss[0]))
			if ipNet == nil {
				continue
			}
			route.Net = *ipNet
			route.Gateway = net.ParseIP(ss[1])
			if route.Gateway == nil {
				route.Gateway = gw
			}

			config.Routes = append(config.Routes, route)
		}
	}

	l.md.config = config

	return
}
