package grpc

import (
	mdata "github.com/go-gost/gost/pkg/metadata"
)

const (
	defaultBacklog = 128
)

type metadata struct {
	backlog  int
	insecure bool
}

func (l *grpcListener) parseMetadata(md mdata.Metadata) (err error) {
	const (
		backlog  = "backlog"
		insecure = "grpcInsecure"
	)

	l.md.backlog = mdata.GetInt(md, backlog)
	if l.md.backlog <= 0 {
		l.md.backlog = defaultBacklog
	}

	l.md.insecure = mdata.GetBool(md, insecure)
	return
}
