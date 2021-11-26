package sni

import (
	"time"

	md "github.com/go-gost/gost/pkg/metadata"
)

type metadata struct {
	host           string
	connectTimeout time.Duration
}

func (c *sniConnector) parseMetadata(md md.Metadata) (err error) {
	const (
		host           = "host"
		connectTimeout = "timeout"
	)

	c.md.host = md.GetString(host)
	c.md.connectTimeout = md.GetDuration(connectTimeout)

	return
}
