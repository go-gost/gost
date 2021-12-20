package tls

import (
	mdata "github.com/go-gost/gost/pkg/metadata"
)

type metadata struct {
	host string
}

func (d *obfsTLSDialer) parseMetadata(md mdata.Metadata) (err error) {
	const (
		host = "host"
	)

	d.md.host = mdata.GetString(md, host)
	return
}
