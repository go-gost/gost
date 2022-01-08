package rtcp

import (
	"time"

	mdata "github.com/go-gost/gost/pkg/metadata"
)

const (
	defaultKeepAlivePeriod = 180 * time.Second
	defaultBacklog         = 128
)

type metadata struct {
}

func (l *rtcpListener) parseMetadata(md mdata.Metadata) (err error) {
	return
}
