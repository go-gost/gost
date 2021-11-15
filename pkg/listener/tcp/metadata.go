package tcp

import (
	"time"

	md "github.com/go-gost/gost/pkg/metadata"
)

const (
	defaultKeepAlivePeriod = 180 * time.Second
)

type metadata struct {
	keepAlive       bool
	keepAlivePeriod time.Duration
}

func (l *tcpListener) parseMetadata(md md.Metadata) (err error) {
	const (
		keepAlive       = "keepAlive"
		keepAlivePeriod = "keepAlivePeriod"
	)

	l.md.keepAlive = md.GetBool(keepAlive)
	l.md.keepAlivePeriod = md.GetDuration(keepAlivePeriod)

	return
}
