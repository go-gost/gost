package v4

import (
	"time"

	mdata "github.com/go-gost/gost/v3/pkg/metadata"
)

type metadata struct {
	connectTimeout time.Duration
	disable4a      bool
}

func (c *socks4Connector) parseMetadata(md mdata.Metadata) (err error) {
	const (
		connectTimeout = "timeout"
		disable4a      = "disable4a"
	)

	c.md.connectTimeout = mdata.GetDuration(md, connectTimeout)
	c.md.disable4a = mdata.GetBool(md, disable4a)

	return
}
