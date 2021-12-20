package rtcp

import (
	"time"

	mdata "github.com/go-gost/gost/pkg/metadata"
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

func (l *rtcpListener) parseMetadata(md mdata.Metadata) (err error) {
	const (
		enableMux  = "mux"
		backlog    = "backlog"
		retryCount = "retry"
	)

	l.md.enableMux = mdata.GetBool(md, enableMux)
	l.md.retryCount = mdata.GetInt(md, retryCount)

	l.md.backlog = mdata.GetInt(md, backlog)
	if l.md.backlog <= 0 {
		l.md.backlog = defaultBacklog
	}
	return
}
