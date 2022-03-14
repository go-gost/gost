package h2

import (
	mdata "github.com/go-gost/gost/v3/pkg/metadata"
)

const (
	defaultBacklog = 128
)

type metadata struct {
	path    string
	backlog int
}

func (l *h2Listener) parseMetadata(md mdata.Metadata) (err error) {
	const (
		path    = "path"
		backlog = "backlog"
	)

	l.md.backlog = mdata.GetInt(md, backlog)
	if l.md.backlog <= 0 {
		l.md.backlog = defaultBacklog
	}

	l.md.path = mdata.GetString(md, path)
	return
}
