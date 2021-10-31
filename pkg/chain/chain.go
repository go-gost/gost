package chain

type Chain struct {
	groups []*NodeGroup
}

func (c *Chain) AddNodeGroup(group *NodeGroup) {
	c.groups = append(c.groups, group)
}

func (c *Chain) GetRouteFor(addr string) (r *Route) {
	if c == nil || len(c.groups) == 0 {
		return
	}

	r = &Route{}
	for _, group := range c.groups {
		node := group.Next()
		if node == nil {
			return
		}
		if node.bypass != nil && node.bypass.Contains(addr) {
			break
		}

		if node.transport.IsMultiplex() {
			tr := node.transport.Copy().WithRoute(r)
			node = node.Copy().WithTransport(tr)
			r = &Route{}
		}

		r.AddNode(node)
	}
	return r
}
