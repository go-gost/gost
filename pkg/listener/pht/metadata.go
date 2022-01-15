package pht

import (
	mdata "github.com/go-gost/gost/pkg/metadata"
)

const (
	defaultBacklog = 128
)

type metadata struct {
	path    string
	backlog int
}

func (l *phtListener) parseMetadata(md mdata.Metadata) (err error) {
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
