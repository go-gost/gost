package quic

import (
	"time"

	mdata "github.com/go-gost/gost/pkg/metadata"
)

type metadata struct {
	keepAlive        bool
	maxIdleTimeout   time.Duration
	handshakeTimeout time.Duration
}

func (d *icmpDialer) parseMetadata(md mdata.Metadata) (err error) {
	const (
		keepAlive        = "keepAlive"
		handshakeTimeout = "handshakeTimeout"
		maxIdleTimeout   = "maxIdleTimeout"
	)

	d.md.handshakeTimeout = mdata.GetDuration(md, handshakeTimeout)

	d.md.keepAlive = mdata.GetBool(md, keepAlive)
	d.md.handshakeTimeout = mdata.GetDuration(md, handshakeTimeout)
	d.md.maxIdleTimeout = mdata.GetDuration(md, maxIdleTimeout)

	return
}
