package bypass

import (
	"net"
	"strconv"

	"github.com/go-gost/gost/v3/pkg/common/matcher"
	"github.com/go-gost/gost/v3/pkg/logger"
)

// Bypass is a filter of address (IP or domain).
type Bypass interface {
	// Contains reports whether the bypass includes addr.
	Contains(addr string) bool
}

type options struct {
	logger logger.Logger
}

type Option func(opts *options)

func LoggerOption(logger logger.Logger) Option {
	return func(opts *options) {
		opts.logger = logger
	}
}

type bypass struct {
	matchers []matcher.Matcher
	reversed bool
	options  options
}

// NewBypass creates and initializes a new Bypass using matchers as its match rules.
// The rules will be reversed if the reversed is true.
func NewBypass(reversed bool, matchers []matcher.Matcher, opts ...Option) Bypass {
	options := options{}
	for _, opt := range opts {
		opt(&options)
	}
	return &bypass{
		matchers: matchers,
		reversed: reversed,
		options:  options,
	}
}

// NewBypassPatterns creates and initializes a new Bypass using matcher patterns as its match rules.
// The rules will be reversed if the reverse is true.
func NewBypassPatterns(reversed bool, patterns []string, opts ...Option) Bypass {
	var matchers []matcher.Matcher
	for _, pattern := range patterns {
		if m := matcher.NewMatcher(pattern); m != nil {
			matchers = append(matchers, m)
		}
	}
	return NewBypass(reversed, matchers, opts...)
}

func (bp *bypass) Contains(addr string) bool {
	if addr == "" || bp == nil || len(bp.matchers) == 0 {
		return false
	}

	// try to strip the port
	if host, port, _ := net.SplitHostPort(addr); host != "" && port != "" {
		if p, _ := strconv.Atoi(port); p > 0 { // port is valid
			addr = host
		}
	}

	var matched bool
	for _, matcher := range bp.matchers {
		if matcher == nil {
			continue
		}
		if matcher.Match(addr) {
			matched = true
			break
		}
	}

	b := !bp.reversed && matched ||
		bp.reversed && !matched
	if b {
		bp.options.logger.Debugf("bypass: %s", addr)
	}
	return b
}
