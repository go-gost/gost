package gost

import (
	"errors"
)

var (
	// ErrInvalidNode is an error that implies the node is invalid.
	ErrInvalidNode = errors.New("invalid node")
)

type Node interface {
	Addr() string
	Protocol() string
	Transport() string
	Params() Params
}

type Params interface {
	Values() map[string]interface{}
}

type clientNode struct {
	node   Node
	client *Client
}

// NodeGroup is a group of nodes.
type NodeGroup struct {
	nodes []*clientNode
}

// AddNodes appends node or node list into group.
func (group *NodeGroup) AddNode(nodes ...Node) error {
	if group == nil {
		return nil
	}
	for _, node := range nodes {
		dialer, err := NewDialer(node.Protocol(), node.Params())
		if err != nil {
			return err
		}
		connector, err := NewConnector(node.Transport(), node.Params())
		if err != nil {
			return err
		}
		group.nodes = append(group.nodes, &clientNode{
			node: node,
			client: &Client{
				Connector: connector,
				Dialer:    dialer,
			},
		})
	}
	return nil
}
