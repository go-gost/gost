package ss

import (
	"time"

	mdata "github.com/go-gost/gost/pkg/metadata"
)

type metadata struct {
	key            string
	connectTimeout time.Duration
	noDelay        bool
}

func (c *ssConnector) parseMetadata(md mdata.Metadata) (err error) {
	const (
		key            = "key"
		connectTimeout = "timeout"
		noDelay        = "nodelay"
	)

	c.md.key = mdata.GetString(md, key)
	c.md.connectTimeout = mdata.GetDuration(md, connectTimeout)
	c.md.noDelay = mdata.GetBool(md, noDelay)

	return
}
