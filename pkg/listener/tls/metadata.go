package tls

import (
	"crypto/tls"

	tls_util "github.com/go-gost/gost/pkg/common/util/tls"
	mdata "github.com/go-gost/gost/pkg/metadata"
)

type metadata struct {
	tlsConfig *tls.Config
}

func (l *tlsListener) parseMetadata(md mdata.Metadata) (err error) {
	const (
		certFile = "certFile"
		keyFile  = "keyFile"
		caFile   = "caFile"
	)

	l.md.tlsConfig, err = tls_util.LoadServerConfig(
		mdata.GetString(md, certFile),
		mdata.GetString(md, keyFile),
		mdata.GetString(md, caFile),
	)
	if err != nil {
		return
	}

	return
}
