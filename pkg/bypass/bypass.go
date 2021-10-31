package bypass

import (
	"net"
	"strconv"
	"strings"

	glob "github.com/gobwas/glob"
)

// Matcher is a generic pattern matcher,
// it gives the match result of the given pattern for specific v.
type Matcher interface {
	Match(v string) bool
}

// NewMatcher creates a Matcher for the given pattern.
// The acutal Matcher depends on the pattern:
// IP Matcher if pattern is a valid IP address.
// CIDR Matcher if pattern is a valid CIDR address.
// Domain Matcher if both of the above are not.
func NewMatcher(pattern string) Matcher {
	if pattern == "" {
		return nil
	}
	if ip := net.ParseIP(pattern); ip != nil {
		return IPMatcher(ip)
	}
	if _, inet, err := net.ParseCIDR(pattern); err == nil {
		return CIDRMatcher(inet)
	}
	return DomainMatcher(pattern)
}

type ipMatcher struct {
	ip net.IP
}

// IPMatcher creates a Matcher for a specific IP address.
func IPMatcher(ip net.IP) Matcher {
	return &ipMatcher{
		ip: ip,
	}
}

func (m *ipMatcher) Match(ip string) bool {
	if m == nil {
		return false
	}
	return m.ip.Equal(net.ParseIP(ip))
}

type cidrMatcher struct {
	ipNet *net.IPNet
}

// CIDRMatcher creates a Matcher for a specific CIDR notation IP address.
func CIDRMatcher(inet *net.IPNet) Matcher {
	return &cidrMatcher{
		ipNet: inet,
	}
}

func (m *cidrMatcher) Match(ip string) bool {
	if m == nil || m.ipNet == nil {
		return false
	}
	return m.ipNet.Contains(net.ParseIP(ip))
}

type domainMatcher struct {
	pattern string
	glob    glob.Glob
}

// DomainMatcher creates a Matcher for a specific domain pattern,
// the pattern can be a plain domain such as 'example.com',
// a wildcard such as '*.exmaple.com' or a special wildcard '.example.com'.
func DomainMatcher(pattern string) Matcher {
	p := pattern
	if strings.HasPrefix(pattern, ".") {
		p = pattern[1:] // trim the prefix '.'
		pattern = "*" + p
	}
	return &domainMatcher{
		pattern: p,
		glob:    glob.MustCompile(pattern),
	}
}

func (m *domainMatcher) Match(domain string) bool {
	if m == nil || m.glob == nil {
		return false
	}

	if domain == m.pattern {
		return true
	}
	return m.glob.Match(domain)
}

// Bypass is a filter of address (IP or domain).
type Bypass interface {
	// Contains reports whether the bypass includes addr.
	Contains(addr string) bool
}

type bypass struct {
	matchers []Matcher
	reversed bool
}

// NewBypass creates and initializes a new Bypass using matchers as its match rules.
// The rules will be reversed if the reversed is true.
func NewBypass(reversed bool, matchers ...Matcher) Bypass {
	return &bypass{
		matchers: matchers,
		reversed: reversed,
	}
}

// NewBypassPatterns creates and initializes a new Bypass using matcher patterns as its match rules.
// The rules will be reversed if the reverse is true.
func NewBypassPatterns(reversed bool, patterns ...string) Bypass {
	var matchers []Matcher
	for _, pattern := range patterns {
		if m := NewMatcher(pattern); m != nil {
			matchers = append(matchers, m)
		}
	}
	return NewBypass(reversed, matchers...)
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
	return !bp.reversed && matched ||
		bp.reversed && !matched
}
