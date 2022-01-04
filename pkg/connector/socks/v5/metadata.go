package v5

import (
	"time"

	mdata "github.com/go-gost/gost/pkg/metadata"
)

type metadata struct {
	connectTimeout time.Duration
	noTLS          bool
}

func (c *socks5Connector) parseMetadata(md mdata.Metadata) (err error) {
	const (
		connectTimeout = "timeout"
		noTLS          = "notls"
	)

	c.md.connectTimeout = mdata.GetDuration(md, connectTimeout)
	c.md.noTLS = mdata.GetBool(md, noTLS)

	return
}
