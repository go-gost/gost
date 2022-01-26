package chain

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
		if node.Bypass != nil && node.Bypass.Contains(address) {
			break
		}

		if node.Transport.Multiplex() {
			tr := node.Transport.Copy().
				WithRoute(r)
			node = node.Copy()
			node.Transport = tr
			r = &route{}
		}

		r.AddNode(node)
	}
	return r
}

func (c *Chain) IsEmpty() bool {
	return c == nil || len(c.groups) == 0
}
