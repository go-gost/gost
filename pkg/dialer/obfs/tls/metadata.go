package tls

import (
	md "github.com/go-gost/gost/pkg/metadata"
)

type metadata struct {
	host string
}

func (d *obfsTLSDialer) parseMetadata(md md.Metadata) (err error) {
	const (
		host = "host"
	)

	d.md.host = md.GetString(host)
	return
}
