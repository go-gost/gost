package udp

import (
	"time"

	md "github.com/go-gost/gost/pkg/metadata"
)

const (
	defaultTTL            = 60 * time.Second
	defaultReadBufferSize = 4096
	defaultReadQueueSize  = 128
	defaultConnQueueSize  = 128
)

type metadata struct {
	ttl time.Duration

	readBufferSize int
	readQueueSize  int
	connQueueSize  int
}

func (l *udpListener) parseMetadata(md md.Metadata) (err error) {
	const (
		ttl            = "ttl"
		readBufferSize = "readBufferSize"
		readQueueSize  = "readQueueSize"
		connQueueSize  = "connQueueSize"
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

	l.md.connQueueSize = md.GetInt(connQueueSize)
	if l.md.connQueueSize <= 0 {
		l.md.connQueueSize = defaultConnQueueSize
	}

	return
}
