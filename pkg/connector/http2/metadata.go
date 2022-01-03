package http2

import (
	"time"

	mdata "github.com/go-gost/gost/pkg/metadata"
)

const (
	defaultUserAgent = "Chrome/78.0.3904.106"
)

type metadata struct {
	connectTimeout time.Duration
	UserAgent      string
}

func (c *http2Connector) parseMetadata(md mdata.Metadata) (err error) {
	const (
		connectTimeout = "timeout"
		userAgent      = "userAgent"
	)

	c.md.connectTimeout = mdata.GetDuration(md, connectTimeout)
	c.md.UserAgent = mdata.GetString(md, userAgent)
	if c.md.UserAgent == "" {
		c.md.UserAgent = defaultUserAgent
	}

	return
}
