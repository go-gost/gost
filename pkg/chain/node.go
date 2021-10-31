package chain

import (
	"sync"
	"time"

	"github.com/go-gost/gost/pkg/bypass"
)

type Node struct {
	name      string
	addr      string
	transport *Transport
	bypass    bypass.Bypass
	marker    *failMarker
}

func NewNode(name, addr string) *Node {
	return &Node{
		name:   name,
		addr:   addr,
		marker: &failMarker{},
	}
}

func (node *Node) Name() string {
	return node.name
}

func (node *Node) Addr() string {
	return node.addr
}

func (node *Node) WithTransport(tr *Transport) *Node {
	node.transport = tr
	return node
}

func (node *Node) WithBypass(bp bypass.Bypass) *Node {
	node.bypass = bp
	return node
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

	selector := g.selector
	if selector == nil {
		return g.nodes[0]
	}

	return selector.Select(g.nodes...)
}

type failMarker struct {
	failTime  int64
	failCount uint32
	mux       sync.RWMutex
}

func (m *failMarker) FailTime() int64 {
	if m == nil {
		return 0
	}

	m.mux.RLock()
	defer m.mux.RUnlock()

	return m.failTime
}

func (m *failMarker) FailCount() uint32 {
	if m == nil {
		return 0
	}

	m.mux.RLock()
	defer m.mux.RUnlock()

	return m.failCount
}

func (m *failMarker) Mark() {
	if m == nil {
		return
	}

	m.mux.Lock()
	defer m.mux.Unlock()

	m.failTime = time.Now().Unix()
	m.failCount++
}

func (m *failMarker) Reset() {
	if m == nil {
		return
	}

	m.mux.Lock()
	defer m.mux.Unlock()

	m.failTime = 0
	m.failCount = 0
}
