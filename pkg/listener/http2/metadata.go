package http2

import (
	mdata "github.com/go-gost/gost/v3/pkg/metadata"
)

const (
	defaultBacklog = 128
)

type metadata struct {
	backlog int
}

func (l *http2Listener) parseMetadata(md mdata.Metadata) (err error) {
	const (
		backlog = "backlog"
	)

	l.md.backlog = mdata.GetInt(md, backlog)
	if l.md.backlog <= 0 {
		l.md.backlog = defaultBacklog
	}
	return
}
