package http

import (
	md "github.com/go-gost/gost/pkg/metadata"
)

const (
	keepAlive       = "keepAlive"
	keepAlivePeriod = "keepAlivePeriod"
)

type metadata struct {
}

func (l *obfsListener) parseMetadata(md md.Metadata) (err error) {
	return
}
