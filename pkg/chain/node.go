package chain

import (
	"sync/atomic"
	"time"

	"github.com/go-gost/gost/pkg/bypass"
	"github.com/go-gost/gost/pkg/hosts"
	"github.com/go-gost/gost/pkg/resolver"
)

type Node struct {
	Name      string
	Addr      string
	Transport *Transport
	Bypass    bypass.Bypass
	Resolver  resolver.Resolver
	Hosts     hosts.HostMapper
	Marker    *FailMarker
}

func (node *Node) Copy() *Node {
	n := &Node{}
	*n = *node
	return n
}

type NodeGroup struct {
	nodes    []*Node
	selector Selector
}

func NewNodeGroup(nodes ...*Node) *NodeGroup {
	return &NodeGroup{
		nodes: nodes,
	}
}

func (g *NodeGroup) AddNode(node *Node) {
	g.nodes = append(g.nodes, node)
}

func (g *NodeGroup) WithSelector(selector Selector) *NodeGroup {
	g.selector = selector
	return g
}

func (g *NodeGroup) Next() *Node {
	if g == nil || len(g.nodes) == 0 {
		return nil
	}

	s := g.selector
	if s == nil {
		s = DefaultSelector
	}

	return s.Select(g.nodes...)
}

type FailMarker struct {
	failTime  int64
	failCount int64
}

func (m *FailMarker) FailTime() int64 {
	if m == nil {
		return 0
	}

	return atomic.LoadInt64(&m.failTime)
}

func (m *FailMarker) FailCount() int64 {
	if m == nil {
		return 0
	}

	return atomic.LoadInt64(&m.failCount)
}

func (m *FailMarker) Mark() {
	if m == nil {
		return
	}

	atomic.AddInt64(&m.failCount, 1)
	atomic.StoreInt64(&m.failTime, time.Now().Unix())
}

func (m *FailMarker) Reset() {
	if m == nil {
		return
	}

	atomic.StoreInt64(&m.failCount, 0)
}
