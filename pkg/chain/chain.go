package chain

type Chain struct {
	Name   string
	groups []*NodeGroup
}

func (c *Chain) AddNodeGroup(group *NodeGroup) {
	c.groups = append(c.groups, group)
}

func (c *Chain) GetRoute() (r *Route) {
	if c == nil || len(c.groups) == 0 {
		return
	}

	r = &Route{}
	for _, group := range c.groups {
		node := group.Next()
		if node == nil {
			return
		}
		// TODO: bypass

		if node.Transport().IsMultiplex() {
			tr := node.Transport().WithRoute(r)
			node = node.WithTransport(tr)
			r = &Route{}
		}

		r.AddNode(node)
	}
	return r
}
