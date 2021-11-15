package rtcp

import (
	"time"

	md "github.com/go-gost/gost/pkg/metadata"
)

const (
	defaultKeepAlivePeriod = 180 * time.Second
	defaultConnQueueSize   = 128
)

type metadata struct {
	enableMux     bool
	connQueueSize int
	retryCount    int
}

func (l *rtcpListener) parseMetadata(md md.Metadata) (err error) {
	const (
		enableMux     = "mux"
		connQueueSize = "connQueueSize"
		retryCount    = "retry"
	)

	l.md.enableMux = md.GetBool(enableMux)
	l.md.retryCount = md.GetInt(retryCount)

	l.md.connQueueSize = md.GetInt(connQueueSize)
	if l.md.connQueueSize <= 0 {
		l.md.connQueueSize = defaultConnQueueSize
	}
	return
}
