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
	DefaultMaxFails    = 1
	DefaultFailTimeout = 30 * time.Second
)

var (
	defaultSelector Selector = NewSelector(nil)
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

// RoundStrategy is a strategy for node selector.
// The node will be selected by round-robin algorithm.
type RoundRobinStrategy struct {
	counter uint64
}

func (s *RoundRobinStrategy) Apply(nodes ...*Node) *Node {
	if len(nodes) == 0 {
		return nil
	}

	n := atomic.AddUint64(&s.counter, 1) - 1
	return nodes[int(n%uint64(len(nodes)))]
}

// RandomStrategy is a strategy for node selector.
// The node will be selected randomly.
type RandomStrategy struct {
	Seed int64
	rand *rand.Rand
	once sync.Once
	mux  sync.Mutex
}

func (s *RandomStrategy) Apply(nodes ...*Node) *Node {
	s.once.Do(func() {
		seed := s.Seed
		if seed == 0 {
			seed = time.Now().UnixNano()
		}
		s.rand = rand.New(rand.NewSource(seed))
	})
	if len(nodes) == 0 {
		return nil
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	r := s.rand.Int()

	return nodes[r%len(nodes)]
}

// FIFOStrategy is a strategy for node selector.
// The node will be selected from first to last,
// and will stick to the selected node until it is failed.
type FIFOStrategy struct{}

// Apply applies the fifo strategy for the nodes.
func (s *FIFOStrategy) Apply(nodes ...*Node) *Node {
	if len(nodes) == 0 {
		return nil
	}
	return nodes[0]
}

type Filter interface {
	Filter(nodes ...*Node) []*Node
}

// FailFilter filters the dead node.
// A node is marked as dead if its failed count is greater than MaxFails.
type FailFilter struct {
	MaxFails    int
	FailTimeout time.Duration
}

// Filter filters dead nodes.
func (f *FailFilter) Filter(nodes ...*Node) []*Node {
	maxFails := f.MaxFails
	if maxFails == 0 {
		maxFails = DefaultMaxFails
	}
	failTimeout := f.FailTimeout
	if failTimeout == 0 {
		failTimeout = DefaultFailTimeout
	}

	if len(nodes) <= 1 || maxFails < 0 {
		return nodes
	}
	var nl []*Node
	for _, node := range nodes {
		if node.marker.FailCount() < uint32(maxFails) ||
			time.Since(time.Unix(node.marker.FailTime(), 0)) >= failTimeout {
			nl = append(nl, node)
		}
	}
	return nl
}

// InvalidFilter filters the invalid node.
// A node is invalid if its port is invalid (negative or zero value).
type InvalidFilter struct{}

// Filter filters invalid nodes.
func (f *InvalidFilter) Filter(nodes ...*Node) []*Node {
	var nl []*Node
	for _, node := range nodes {
		_, sport, _ := net.SplitHostPort(node.Addr())
		if port, _ := strconv.Atoi(sport); port > 0 {
			nl = append(nl, node)
		}
	}
	return nl
}
