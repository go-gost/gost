package tcp

import (
	"time"

	md "github.com/go-gost/gost/pkg/metadata"
)

const (
	dialTimeout = "dialTimeout"
)

const (
	defaultDialTimeout = 5 * time.Second
)

type metadata struct {
	dialTimeout time.Duration
}

func (d *tcpDialer) parseMetadata(md md.Metadata) (err error) {
	return
}
