package tcp

import (
	md "github.com/go-gost/gost/pkg/metadata"
)

type metadata struct {
}

func (l *tcpListener) parseMetadata(md md.Metadata) (err error) {
	return
}
