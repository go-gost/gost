package tun

import (
	"net"
	"strings"

	md "github.com/go-gost/gost/pkg/metadata"
)

const (
	DefaultMTU = 1350
)

type metadata struct {
	name string
	net  string
	// peer addr of point-to-point on MacOS
	peer   string
	mtu    int
	routes []ipRoute
	// default gateway
	gateway string
	tcp     bool
}

func (l *tunListener) parseMetadata(md md.Metadata) (err error) {
	const (
		name    = "name"
		netKey  = "net"
		peer    = "peer"
		mtu     = "mtu"
		routes  = "routes"
		gateway = "gw"
		tcp     = "tcp"
	)

	l.md.name = md.GetString(name)
	l.md.net = md.GetString(netKey)
	l.md.peer = md.GetString(peer)
	l.md.mtu = md.GetInt(mtu)

	if l.md.mtu <= 0 {
		l.md.mtu = DefaultMTU
	}

	l.md.gateway = md.GetString(gateway)
	l.md.tcp = md.GetBool(tcp)

	gw := net.ParseIP(l.md.gateway)

	for _, s := range md.GetStrings(routes) {
		ss := strings.SplitN(s, " ", 2)
		if len(ss) == 2 {
			var route ipRoute
			_, ipNet, _ := net.ParseCIDR(strings.TrimSpace(ss[0]))
			if ipNet == nil {
				continue
			}
			route.Dest = *ipNet
			route.Gateway = net.ParseIP(ss[1])
			if route.Gateway == nil {
				route.Gateway = gw
			}

			l.md.routes = append(l.md.routes, route)
		}
	}

	return
}
