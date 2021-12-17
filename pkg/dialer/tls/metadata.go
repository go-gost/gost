package tls

import (
	"crypto/tls"
	"net"
	"time"

	tls_util "github.com/go-gost/gost/pkg/common/util/tls"
	md "github.com/go-gost/gost/pkg/metadata"
)

type metadata struct {
	tlsConfig        *tls.Config
	handshakeTimeout time.Duration
}

func (d *tlsDialer) parseMetadata(md md.Metadata) (err error) {
	const (
		certFile   = "certFile"
		keyFile    = "keyFile"
		caFile     = "caFile"
		secure     = "secure"
		serverName = "serverName"

		handshakeTimeout = "handshakeTimeout"
	)

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

	d.md.handshakeTimeout = md.GetDuration(handshakeTimeout)

	return
}
