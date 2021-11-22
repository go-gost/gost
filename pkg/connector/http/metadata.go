package http

import (
	"net/url"
	"strings"
	"time"

	md "github.com/go-gost/gost/pkg/metadata"
)

const (
	defaultUserAgent = "Chrome/78.0.3904.106"
)

type metadata struct {
	connectTimeout time.Duration
	UserAgent      string
	User           *url.Userinfo
}

func (c *httpConnector) parseMetadata(md md.Metadata) (err error) {
	const (
		connectTimeout = "timeout"
		userAgent      = "userAgent"
		user           = "user"
	)

	c.md.connectTimeout = md.GetDuration(connectTimeout)
	c.md.UserAgent, _ = md.Get(userAgent).(string)
	if c.md.UserAgent == "" {
		c.md.UserAgent = defaultUserAgent
	}

	if v := md.GetString(user); v != "" {
		ss := strings.SplitN(v, ":", 2)
		if len(ss) == 1 {
			c.md.User = url.User(ss[0])
		} else {
			c.md.User = url.UserPassword(ss[0], ss[1])
		}
	}

	return
}
