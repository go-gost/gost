package tls

import (
	"crypto/tls"
	"time"

	tls_util "github.com/go-gost/gost/pkg/common/util/tls"
	md "github.com/go-gost/gost/pkg/metadata"
)

type metadata struct {
	tlsConfig       *tls.Config
	keepAlivePeriod time.Duration
}

func (l *tlsListener) parseMetadata(md md.Metadata) (err error) {
	const (
		certFile        = "certFile"
		keyFile         = "keyFile"
		caFile          = "caFile"
		keepAlivePeriod = "keepAlivePeriod"
	)

	if md.GetString(certFile) != "" ||
		md.GetString(keyFile) != "" ||
		md.GetString(caFile) != "" {
		l.md.tlsConfig, err = tls_util.LoadTLSConfig(
			md.GetString(certFile),
			md.GetString(keyFile),
			md.GetString(caFile),
		)
		if err != nil {
			return
		}
	} else {
		l.md.tlsConfig = tls_util.DefaultConfig
	}

	l.md.keepAlivePeriod = md.GetDuration(keepAlivePeriod)
	return
}
