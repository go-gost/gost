package chain

import (
	"math/rand"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// default options for FailFilter
const (
	DefaultFailTimeout = 30 * time.Second
)

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

type Strategy interface {
	Apply(nodes ...*Node) *Node
}

type roundRobinStrategy struct {
	counter uint64
}

// RoundRobinStrategy is a strategy for node selector.
// The node will be selected by round-robin algorithm.
func RoundRobinStrategy() Strategy {
	return &roundRobinStrategy{}
}

func (s *roundRobinStrategy) Apply(nodes ...*Node) *Node {
	if len(nodes) == 0 {
		return nil
	}

	n := atomic.AddUint64(&s.counter, 1) - 1
	return nodes[int(n%uint64(len(nodes)))]
}

type randomStrategy struct {
	rand *rand.Rand
	mux  sync.Mutex
}

// RandomStrategy is a strategy for node selector.
// The node will be selected randomly.
func RandomStrategy() Strategy {
	return &randomStrategy{
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (s *randomStrategy) Apply(nodes ...*Node) *Node {
	if len(nodes) == 0 {
		return nil
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	r := s.rand.Int()

	return nodes[r%len(nodes)]
}

type fifoStrategy struct{}

// FIFOStrategy is a strategy for node selector.
// The node will be selected from first to last,
// and will stick to the selected node until it is failed.
func FIFOStrategy() Strategy {
	return &fifoStrategy{}
}

// Apply applies the fifo strategy for the nodes.
func (s *fifoStrategy) Apply(nodes ...*Node) *Node {
	if len(nodes) == 0 {
		return nil
	}
	return nodes[0]
}

type Filter interface {
	Filter(nodes ...*Node) []*Node
}

type failFilter struct {
	maxFails    int
	failTimeout time.Duration
}

// FailFilter filters the dead node.
// A node is marked as dead if its failed count is greater than MaxFails.
func FailFilter(maxFails int, timeout time.Duration) Filter {
	return &failFilter{
		maxFails:    maxFails,
		failTimeout: timeout,
	}
}

// Filter filters dead nodes.
func (f *failFilter) Filter(nodes ...*Node) []*Node {
	maxFails := f.maxFails
	failTimeout := f.failTimeout
	if failTimeout == 0 {
		failTimeout = DefaultFailTimeout
	}

	if len(nodes) <= 1 || maxFails <= 0 {
		return nodes
	}
	var nl []*Node
	for _, node := range nodes {
		if node.Marker().FailCount() < int64(maxFails) ||
			time.Since(time.Unix(node.Marker().FailTime(), 0)) >= failTimeout {
			nl = append(nl, node)
		}
	}
	return nl
}

type invalidFilter struct{}

// InvalidFilter filters the invalid node.
// A node is invalid if its port is invalid (negative or zero value).
func InvalidFilter() Filter {
	return &invalidFilter{}
}

// Filter filters invalid nodes.
func (f *invalidFilter) Filter(nodes ...*Node) []*Node {
	var nl []*Node
	for _, node := range nodes {
		_, sport, _ := net.SplitHostPort(node.Addr())
		if port, _ := strconv.Atoi(sport); port > 0 {
			nl = append(nl, node)
		}
	}
	return nl
}
