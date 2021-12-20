package udp

import (
	"time"

	mdata "github.com/go-gost/gost/pkg/metadata"
)

const (
	defaultTTL            = 60 * time.Second
	defaultReadBufferSize = 1024
)

type metadata struct {
	ttl            time.Duration
	readBufferSize int
}

func (l *redirectListener) parseMetadata(md mdata.Metadata) (err error) {
	const (
		ttl            = "ttl"
		readBufferSize = "readBufferSize"
	)

	l.md.ttl = mdata.GetDuration(md, ttl)
	if l.md.ttl <= 0 {
		l.md.ttl = defaultTTL
	}

	l.md.readBufferSize = mdata.GetInt(md, readBufferSize)
	if l.md.readBufferSize <= 0 {
		l.md.readBufferSize = defaultReadBufferSize
	}

	return
}
