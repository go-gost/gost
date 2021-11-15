package kcp

import md "github.com/go-gost/gost/pkg/metadata"

const (
	defaultQueueSize = 128
)

type metadata struct {
	config *Config

	connQueueSize int
}

func (l *kcpListener) parseMetadata(md md.Metadata) (err error) {
	const (
		connQueueSize = "connQueueSize"
	)

	l.md.connQueueSize = md.GetInt(connQueueSize)

	return
}
