package rudp

import (
	"time"

	mdata "github.com/go-gost/gost/v3/pkg/metadata"
)

const (
	defaultTTL            = 5 * time.Second
	defaultReadBufferSize = 1024
	defaultReadQueueSize  = 128
	defaultBacklog        = 128
)

type metadata struct {
	ttl            time.Duration
	readBufferSize int
	readQueueSize  int
	backlog        int
	retryCount     int
}

func (l *rudpListener) parseMetadata(md mdata.Metadata) (err error) {
	const (
		ttl            = "ttl"
		readBufferSize = "readBufferSize"
		readQueueSize  = "readQueueSize"
		backlog        = "backlog"
		retryCount     = "retry"
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

	l.md.retryCount = mdata.GetInt(md, retryCount)
	return
}
