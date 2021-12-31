package chain

type Chainable interface {
	WithChain(chain *Chain)
}

type Chain struct {
	groups []*NodeGroup
}

func (c *Chain) AddNodeGroup(group *NodeGroup) {
	c.groups = append(c.groups, group)
}

func (c *Chain) GetRoute() (r *route) {
	return c.GetRouteFor("tcp", "")
}

func (c *Chain) GetRouteFor(network, address string) (r *route) {
	if c == nil || len(c.groups) == 0 {
		return
	}

	r = &route{}
	for _, group := range c.groups {
		node := group.Next()
		if node == nil {
			return
		}
		if node.bypass != nil && node.bypass.Contains(address) {
			break
		}

		if node.transport.Multiplex() {
			tr := node.transport.Copy().
				WithRoute(r)
			node = node.Copy().
				WithTransport(tr)
			r = &route{}
		}

		r.AddNode(node)
	}
	return r
}

func (c *Chain) IsEmpty() bool {
	return c == nil || len(c.groups) == 0
}
