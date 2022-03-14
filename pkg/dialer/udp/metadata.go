package udp

import (
	"time"

	md "github.com/go-gost/gost/v3/pkg/metadata"
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

func (d *udpDialer) parseMetadata(md md.Metadata) (err error) {
	return
}
