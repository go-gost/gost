package chain

import (
	"context"
	"errors"
	"net"
)

var (
	ErrEmptyRoute = errors.New("empty route")
)

type Route struct {
	nodes []*Node
}

func (r *Route) AddNode(node *Node) {
	r.nodes = append(r.nodes, node)
}

func (r *Route) Connect(ctx context.Context) (conn net.Conn, err error) {
	if r.IsEmpty() {
		return nil, ErrEmptyRoute
	}

	node := r.nodes[0]
	cc, err := node.transport.Dial(ctx, r.nodes[0].Addr())
	if err != nil {
		node.marker.Mark()
		return
	}

	cn, err := node.transport.Handshake(ctx, cc)
	if err != nil {
		cc.Close()
		node.marker.Mark()
		return
	}
	node.marker.Reset()

	preNode := node
	for _, node := range r.nodes[1:] {
		cc, err = preNode.transport.Connect(ctx, cn, "tcp", node.Addr())
		if err != nil {
			cn.Close()
			node.marker.Mark()
			return
		}
		cc, err = node.transport.Handshake(ctx, cc)
		if err != nil {
			cn.Close()
			node.marker.Mark()
			return
		}
		node.marker.Reset()

		cn = cc
		preNode = node
	}

	conn = cn
	return
}

func (r *Route) Dial(ctx context.Context, network, address string) (net.Conn, error) {
	if r.IsEmpty() {
		return r.dialDirect(ctx, network, address)
	}

	conn, err := r.Connect(ctx)
	if err != nil {
		return nil, err
	}

	cc, err := r.Last().transport.Connect(ctx, conn, network, address)
	if err != nil {
		conn.Close()
		return nil, err
	}
	return cc, nil
}

func (r *Route) dialDirect(ctx context.Context, network, address string) (net.Conn, error) {
	switch network {
	case "udp", "udp4", "udp6":
		if address == "" {
			return net.ListenUDP(network, nil)
		}
	default:
	}

	d := &net.Dialer{}
	return d.DialContext(ctx, network, address)
}

func (r *Route) IsEmpty() bool {
	return r == nil || len(r.nodes) == 0
}

func (r *Route) Last() *Node {
	if r.IsEmpty() {
		return nil
	}
	return r.nodes[len(r.nodes)-1]
}

func (r *Route) Path() (path []*Node) {
	if r == nil || len(r.nodes) == 0 {
		return nil
	}

	for _, node := range r.nodes {
		if node.transport != nil && node.transport.route != nil {
			path = append(path, node.transport.route.Path()...)
		}
		path = append(path, node)
	}
	return
}
