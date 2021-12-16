package tls

import (
	md "github.com/go-gost/gost/pkg/metadata"
)

type metadata struct {
}

func (l *obfsListener) parseMetadata(md md.Metadata) (err error) {
	return
}
