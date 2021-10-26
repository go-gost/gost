package chain

type Node struct {
	name      string
	addr      string
	transport *Transport
}

func NewNode(name, addr string) *Node {
	return &Node{
		name: name,
		addr: addr,
	}
}

func (node *Node) Name() string {
	return node.name
}

func (node *Node) Addr() string {
	return node.addr
}

func (node *Node) Transport() *Transport {
	return node.transport
}

func (node *Node) WithTransport(tr *Transport) *Node {
	node.transport = tr
	return node
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

func (g *NodeGroup) WithSelector(selector Selector) {
	g.selector = selector
}

func (g *NodeGroup) Next() *Node {
	selector := g.selector
	if selector == nil {
		// selector = defaultSelector
		return g.nodes[0]
	}
	return selector.Select(g.nodes...)
}
