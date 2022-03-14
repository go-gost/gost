package h2

import (
	mdata "github.com/go-gost/gost/v3/pkg/metadata"
)

type metadata struct {
	host string
	path string
}

func (d *h2Dialer) parseMetadata(md mdata.Metadata) (err error) {
	const (
		host = "host"
		path = "path"
	)

	d.md.host = mdata.GetString(md, host)
	d.md.path = mdata.GetString(md, path)

	return
}
