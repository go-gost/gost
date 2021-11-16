package rtcp

import (
	"time"

	md "github.com/go-gost/gost/pkg/metadata"
)

const (
	defaultKeepAlivePeriod = 180 * time.Second
	defaultBacklog         = 128
)

type metadata struct {
	enableMux  bool
	backlog    int
	retryCount int
}

func (l *rtcpListener) parseMetadata(md md.Metadata) (err error) {
	const (
		enableMux  = "mux"
		backlog    = "backlog"
		retryCount = "retry"
	)

	l.md.enableMux = md.GetBool(enableMux)
	l.md.retryCount = md.GetInt(retryCount)

	l.md.backlog = md.GetInt(backlog)
	if l.md.backlog <= 0 {
		l.md.backlog = defaultBacklog
	}
	return
}
