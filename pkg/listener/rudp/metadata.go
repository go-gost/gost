package rudp

import (
	"time"

	md "github.com/go-gost/gost/pkg/metadata"
)

const (
	defaultTTL            = 60 * time.Second
	defaultReadBufferSize = 4096
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

func (l *rudpListener) parseMetadata(md md.Metadata) (err error) {
	const (
		ttl            = "ttl"
		readBufferSize = "readBufferSize"
		readQueueSize  = "readQueueSize"
		backlog        = "backlog"
		retryCount     = "retry"
	)

	l.md.ttl = md.GetDuration(ttl)
	if l.md.ttl <= 0 {
		l.md.ttl = defaultTTL
	}
	l.md.readBufferSize = md.GetInt(readBufferSize)
	if l.md.readBufferSize <= 0 {
		l.md.readBufferSize = defaultReadBufferSize
	}

	l.md.readQueueSize = md.GetInt(readQueueSize)
	if l.md.readQueueSize <= 0 {
		l.md.readQueueSize = defaultReadQueueSize
	}

	l.md.backlog = md.GetInt(backlog)
	if l.md.backlog <= 0 {
		l.md.backlog = defaultBacklog
	}

	l.md.retryCount = md.GetInt(retryCount)
	return
}
