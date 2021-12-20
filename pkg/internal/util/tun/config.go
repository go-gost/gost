package tun

import "net"

// Route is an IP routing entry
type Route struct {
	Net     net.IPNet
	Gateway net.IP
}

type Config struct {
	Name string
	Net  string
	// peer addr of point-to-point on MacOS
	Peer    string
	MTU     int
	Gateway string
	Routes  []Route
}
