package listener

import "net"

// Listener is a server listener, just like a net.Listener.
type Listener interface {
	Init(md Metadata) error
	net.Listener
}

// Accepter represents a network endpoint that can accept connection from peer.
type Accepter interface {
	Accept() (net.Conn, error)
}
