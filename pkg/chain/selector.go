package chain

var (
	defaultSelector Selector = NewSelector(nil)
)

type Filter interface {
	Filter(nodes ...*Node) []*Node
	String() string
}

type Strategy interface {
	Apply(nodes ...*Node) *Node
	String() string
}

type Selector interface {
	Select(nodes ...*Node) *Node
}

type selector struct {
	strategy Strategy
	filters  []Filter
}

func NewSelector(strategy Strategy, filters ...Filter) Selector {
	return &selector{
		filters:  filters,
		strategy: strategy,
	}
}

func (s *selector) Select(nodes ...*Node) *Node {
	for _, filter := range s.filters {
		nodes = filter.Filter(nodes...)
	}
	if len(nodes) == 0 {
		return nil
	}
	return s.strategy.Apply(nodes...)
}
