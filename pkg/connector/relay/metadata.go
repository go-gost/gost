package relay

import (
	"time"

	mdata "github.com/go-gost/gost/pkg/metadata"
)

type metadata struct {
	connectTimeout time.Duration
	noDelay        bool
}

func (c *relayConnector) parseMetadata(md mdata.Metadata) (err error) {
	const (
		connectTimeout = "connectTimeout"
		noDelay        = "nodelay"
	)

	c.md.connectTimeout = mdata.GetDuration(md, connectTimeout)
	c.md.noDelay = mdata.GetBool(md, noDelay)

	return
}
