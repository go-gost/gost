package quic

import (
	"crypto/tls"
	"time"

	tls_util "github.com/go-gost/gost/pkg/common/util/tls"
	mdata "github.com/go-gost/gost/pkg/metadata"
)

const (
	defaultBacklog = 128
)

type metadata struct {
	keepAlive        bool
	handshakeTimeout time.Duration
	maxIdleTimeout   time.Duration

	tlsConfig *tls.Config
	cipherKey []byte
	backlog   int
}

func (l *quicListener) parseMetadata(md mdata.Metadata) (err error) {
	const (
		keepAlive        = "keepAlive"
		handshakeTimeout = "handshakeTimeout"
		maxIdleTimeout   = "maxIdleTimeout"

		certFile = "certFile"
		keyFile  = "keyFile"
		caFile   = "caFile"

		backlog   = "backlog"
		cipherKey = "cipherKey"
	)

	l.md.tlsConfig, err = tls_util.LoadServerConfig(
		mdata.GetString(md, certFile),
		mdata.GetString(md, keyFile),
		mdata.GetString(md, caFile),
	)

	if err != nil {
		return
	}
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
