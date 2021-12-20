package http2

import (
	"net/url"
	"strings"
	"time"

	mdata "github.com/go-gost/gost/pkg/metadata"
)

const (
	defaultUserAgent = "Chrome/78.0.3904.106"
)

type metadata struct {
	connectTimeout time.Duration
	UserAgent      string
	User           *url.Userinfo
}

func (c *http2Connector) parseMetadata(md mdata.Metadata) (err error) {
	const (
		connectTimeout = "timeout"
		userAgent      = "userAgent"
		user           = "user"
	)

	c.md.connectTimeout = mdata.GetDuration(md, connectTimeout)
	c.md.UserAgent = mdata.GetString(md, userAgent)
	if c.md.UserAgent == "" {
		c.md.UserAgent = defaultUserAgent
	}

	if v := mdata.GetString(md, user); v != "" {
		ss := strings.SplitN(v, ":", 2)
		if len(ss) == 1 {
			c.md.User = url.User(ss[0])
		} else {
			c.md.User = url.UserPassword(ss[0], ss[1])
		}
	}

	return
}
