package gost

import (
	"context"
	"errors"
	"net"
	"time"
)

var (
	// ErrEmptyRoute is an error that implies the chain is empty.
	ErrEmptyRoute = errors.New("empty route")
)

// Chain is a proxy chain that holds a list of proxy node groups.
type Chain struct {
	groups []*NodeGroup
}

// AddNode appends the node(s) to the chain.
// For each node, it creates a group, and adds the node to the group.
func (c *Chain) AddNode(nodes ...Node) error {
	for _, node := range nodes {
		group := &NodeGroup{}
		if err := group.AddNode(node); err != nil {
			return err
		}
		c.AddNodeGroup(group)
	}
	return nil
}

// AddNodeGroup appends the group(s) to the chain.
func (c *Chain) AddNodeGroup(groups ...*NodeGroup) {
	if c != nil {
		c.groups = append(c.groups, groups...)
	}
}

func (c *Chain) Route() *Route {
	return nil
}

type Route struct {
	nodes []*clientNode
}

// Dial connects to the address on the named network using the provided context.
func (c *Route) Dial(ctx context.Context, network, address string) (conn net.Conn, err error) {
	if len(c.nodes) == 0 {
		switch network {
		case "udp", "udp4", "udp6":
			if address == "" {
				return net.ListenUDP(network, nil)
			}
		default:
		}
		d := &net.Dialer{
			// LocalAddr: laddr, // TODO: optional local address
		}
		return d.DialContext(ctx, network, address)
	}

	cc, err := c.Connect(ctx)
	if err != nil {
		return nil, err
	}

	lastNode := c.nodes[len(c.nodes)-1]
	conn, err = lastNode.client.Connect(ctx, cc, network, address)
	if err != nil {
		cc.Close()
	}
	return
}

// Connect obtains a handshaked connection to the last node of the route.
func (c *Route) Connect(ctx context.Context) (conn net.Conn, err error) {
	if len(c.nodes) == 0 {
		err = ErrEmptyRoute
		return
	}
	nodes := c.nodes
	node := nodes[0]

	cc, err := node.client.Dial(ctx, node.node.Addr())
	if err != nil {
		return
	}

	cn, err := node.client.Handshake(ctx, cc)
	if err != nil {
		cc.Close()
		return
	}

	preNode := node
	for _, node := range nodes[1:] {
		var cc net.Conn
		cc, err = preNode.client.Connect(ctx, cn, "tcp", node.node.Addr())
		if err != nil {
			cn.Close()
			return
		}
		cc, err = node.client.Handshake(ctx, cc)
		if err != nil {
			cn.Close()
			return
		}

		cn = cc
		preNode = node
	}

	conn = cn
	return
}

// ChainOptions holds options for Chain.
type ChainOptions struct {
	Retries int
	Timeout time.Duration
}

// ChainOption allows a common way to set chain options.
type ChainOption func(opts *ChainOptions)

// RetryChainOption specifies the times of retry used by Chain.Dial.
func RetryChainOption(retries int) ChainOption {
	return func(opts *ChainOptions) {
		opts.Retries = retries
	}
}

// TimeoutChainOption specifies the timeout used by Chain.Dial.
func TimeoutChainOption(timeout time.Duration) ChainOption {
	return func(opts *ChainOptions) {
		opts.Timeout = timeout
	}
}
