package udp

import (
	"time"

	mdata "github.com/go-gost/gost/pkg/metadata"
)

const (
	defaultTTL            = 5 * time.Second
	defaultReadBufferSize = 1024
	defaultReadQueueSize  = 128
	defaultBacklog        = 128
)

type metadata struct {
	ttl time.Duration

	readBufferSize int
	readQueueSize  int
	backlog        int
}

func (l *udpListener) parseMetadata(md mdata.Metadata) (err error) {
	const (
		ttl            = "ttl"
		readBufferSize = "readBufferSize"
		readQueueSize  = "readQueueSize"
		backlog        = "backlog"
	)

	l.md.ttl = mdata.GetDuration(md, ttl)
	if l.md.ttl <= 0 {
		l.md.ttl = defaultTTL
	}
	l.md.readBufferSize = mdata.GetInt(md, readBufferSize)
	if l.md.readBufferSize <= 0 {
		l.md.readBufferSize = defaultReadBufferSize
	}

	l.md.readQueueSize = mdata.GetInt(md, readQueueSize)
	if l.md.readQueueSize <= 0 {
		l.md.readQueueSize = defaultReadQueueSize
	}

	l.md.backlog = mdata.GetInt(md, backlog)
	if l.md.backlog <= 0 {
		l.md.backlog = defaultBacklog
	}

	return
}
