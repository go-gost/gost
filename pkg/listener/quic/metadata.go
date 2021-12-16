package quic

import (
	"crypto/tls"
	"time"

	tls_util "github.com/go-gost/gost/pkg/common/util/tls"
	md "github.com/go-gost/gost/pkg/metadata"
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

func (l *quicListener) parseMetadata(md md.Metadata) (err error) {
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
		md.GetString(certFile),
		md.GetString(keyFile),
		md.GetString(caFile),
	)

	if err != nil {
		return
	}
	l.md.backlog = md.GetInt(backlog)
	if l.md.backlog <= 0 {
		l.md.backlog = defaultBacklog
	}

	if key := md.GetString(cipherKey); key != "" {
		l.md.cipherKey = []byte(key)
	}

	l.md.keepAlive = md.GetBool(keepAlive)
	l.md.handshakeTimeout = md.GetDuration(handshakeTimeout)
	l.md.maxIdleTimeout = md.GetDuration(maxIdleTimeout)

	return
}
