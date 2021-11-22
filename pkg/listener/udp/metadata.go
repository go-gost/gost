package udp

import (
	"time"

	md "github.com/go-gost/gost/pkg/metadata"
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

func (l *udpListener) parseMetadata(md md.Metadata) (err error) {
	const (
		ttl            = "ttl"
		readBufferSize = "readBufferSize"
		readQueueSize  = "readQueueSize"
		backlog        = "backlog"
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

	return
}
