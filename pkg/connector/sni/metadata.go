package sni

import (
	"time"

	mdata "github.com/go-gost/gost/pkg/metadata"
)

type metadata struct {
	host           string
	connectTimeout time.Duration
}

func (c *sniConnector) parseMetadata(md mdata.Metadata) (err error) {
	const (
		host           = "host"
		connectTimeout = "timeout"
	)

	c.md.host = mdata.GetString(md, host)
	c.md.connectTimeout = mdata.GetDuration(md, connectTimeout)

	return
}
