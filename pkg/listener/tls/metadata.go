package tls

import (
	"crypto/tls"

	tls_util "github.com/go-gost/gost/pkg/common/util/tls"
	md "github.com/go-gost/gost/pkg/metadata"
)

type metadata struct {
	tlsConfig *tls.Config
}

func (l *tlsListener) parseMetadata(md md.Metadata) (err error) {
	const (
		certFile = "certFile"
		keyFile  = "keyFile"
		caFile   = "caFile"
	)

	l.md.tlsConfig, err = tls_util.LoadServerConfig(
		md.GetString(certFile),
		md.GetString(keyFile),
		md.GetString(caFile),
	)
	if err != nil {
		return
	}

	return
}
