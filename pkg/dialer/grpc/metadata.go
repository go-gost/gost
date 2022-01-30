package grpc

import (
	mdata "github.com/go-gost/gost/pkg/metadata"
)

type metadata struct {
	insecure bool
	host     string
}

func (d *grpcDialer) parseMetadata(md mdata.Metadata) (err error) {
	const (
		insecure = "grpcInsecure"
		host     = "host"
	)

	d.md.insecure = mdata.GetBool(md, insecure)
	d.md.host = mdata.GetString(md, host)

	return
}
