package dns

import (
	"time"

	mdata "github.com/go-gost/gost/pkg/metadata"
)

const (
	defaultBacklog = 128
)

type metadata struct {
	mode           string
	readBufferSize int
	readTimeout    time.Duration
	writeTimeout   time.Duration
	backlog        int
}

func (l *dnsListener) parseMetadata(md mdata.Metadata) (err error) {
	const (
		mode           = "mode"
		readBufferSize = "readBufferSize"

		backlog = "backlog"
	)

	l.md.mode = mdata.GetString(md, mode)
	l.md.readBufferSize = mdata.GetInt(md, readBufferSize)

	l.md.backlog = mdata.GetInt(md, backlog)
	if l.md.backlog <= 0 {
		l.md.backlog = defaultBacklog
	}

	return
}
