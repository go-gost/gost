package quic

import (
	"crypto/tls"
	"net"
	"time"

	tls_util "github.com/go-gost/gost/pkg/common/util/tls"
	md "github.com/go-gost/gost/pkg/metadata"
)

type metadata struct {
	keepAlive        bool
	maxIdleTimeout   time.Duration
	handshakeTimeout time.Duration

	cipherKey []byte
	tlsConfig *tls.Config
}

func (d *quicDialer) parseMetadata(md md.Metadata) (err error) {
	const (
		keepAlive        = "keepAlive"
		handshakeTimeout = "handshakeTimeout"
		maxIdleTimeout   = "maxIdleTimeout"

		certFile   = "certFile"
		keyFile    = "keyFile"
		caFile     = "caFile"
		secure     = "secure"
		serverName = "serverName"

		cipherKey = "cipherKey"
	)

	d.md.handshakeTimeout = md.GetDuration(handshakeTimeout)

	if key := md.GetString(cipherKey); key != "" {
		d.md.cipherKey = []byte(key)
	}

	sn, _, _ := net.SplitHostPort(md.GetString(serverName))
	if sn == "" {
		sn = "localhost"
	}
	d.md.tlsConfig, err = tls_util.LoadClientConfig(
		md.GetString(certFile),
		md.GetString(keyFile),
		md.GetString(caFile),
		md.GetBool(secure),
		sn,
	)

	d.md.keepAlive = md.GetBool(keepAlive)
	d.md.handshakeTimeout = md.GetDuration(handshakeTimeout)
	d.md.maxIdleTimeout = md.GetDuration(maxIdleTimeout)
	return
}
