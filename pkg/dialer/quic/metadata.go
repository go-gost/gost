package quic

import (
	"time"

	mdata "github.com/go-gost/gost/pkg/metadata"
)

type metadata struct {
	keepAlive        bool
	maxIdleTimeout   time.Duration
	handshakeTimeout time.Duration

	cipherKey []byte
	host      string
}

func (d *quicDialer) parseMetadata(md mdata.Metadata) (err error) {
	const (
		keepAlive        = "keepAlive"
		handshakeTimeout = "handshakeTimeout"
		maxIdleTimeout   = "maxIdleTimeout"

		cipherKey = "cipherKey"
		host      = "host"
	)

	d.md.handshakeTimeout = mdata.GetDuration(md, handshakeTimeout)

	if key := mdata.GetString(md, cipherKey); key != "" {
		d.md.cipherKey = []byte(key)
	}

	d.md.keepAlive = mdata.GetBool(md, keepAlive)
	d.md.handshakeTimeout = mdata.GetDuration(md, handshakeTimeout)
	d.md.maxIdleTimeout = mdata.GetDuration(md, maxIdleTimeout)

	d.md.host = mdata.GetString(md, host)
	return
}
