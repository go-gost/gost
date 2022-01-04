package quic

import (
	"time"

	mdata "github.com/go-gost/gost/pkg/metadata"
)

const (
	defaultBacklog = 128
)

type metadata struct {
	keepAlive        bool
	handshakeTimeout time.Duration
	maxIdleTimeout   time.Duration

	cipherKey []byte
	backlog   int
}

func (l *quicListener) parseMetadata(md mdata.Metadata) (err error) {
	const (
		keepAlive        = "keepAlive"
		handshakeTimeout = "handshakeTimeout"
		maxIdleTimeout   = "maxIdleTimeout"

		backlog   = "backlog"
		cipherKey = "cipherKey"
	)

	l.md.backlog = mdata.GetInt(md, backlog)
	if l.md.backlog <= 0 {
		l.md.backlog = defaultBacklog
	}

	if key := mdata.GetString(md, cipherKey); key != "" {
		l.md.cipherKey = []byte(key)
	}

	l.md.keepAlive = mdata.GetBool(md, keepAlive)
	l.md.handshakeTimeout = mdata.GetDuration(md, handshakeTimeout)
	l.md.maxIdleTimeout = mdata.GetDuration(md, maxIdleTimeout)

	return
}
