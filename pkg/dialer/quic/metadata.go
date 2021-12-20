package quic

import (
	"crypto/tls"
	"net"
	"time"

	tls_util "github.com/go-gost/gost/pkg/common/util/tls"
	mdata "github.com/go-gost/gost/pkg/metadata"
)

type metadata struct {
	keepAlive        bool
	maxIdleTimeout   time.Duration
	handshakeTimeout time.Duration

	cipherKey []byte
	tlsConfig *tls.Config
}

func (d *quicDialer) parseMetadata(md mdata.Metadata) (err error) {
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

	d.md.handshakeTimeout = mdata.GetDuration(md, handshakeTimeout)

	if key := mdata.GetString(md, cipherKey); key != "" {
		d.md.cipherKey = []byte(key)
	}

	sn, _, _ := net.SplitHostPort(mdata.GetString(md, serverName))
	if sn == "" {
		sn = "localhost"
	}
	d.md.tlsConfig, err = tls_util.LoadClientConfig(
		mdata.GetString(md, certFile),
		mdata.GetString(md, keyFile),
		mdata.GetString(md, caFile),
		mdata.GetBool(md, secure),
		sn,
	)

	d.md.keepAlive = mdata.GetBool(md, keepAlive)
	d.md.handshakeTimeout = mdata.GetDuration(md, handshakeTimeout)
	d.md.maxIdleTimeout = mdata.GetDuration(md, maxIdleTimeout)
	return
}
