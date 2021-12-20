package http2

import (
	"crypto/tls"
	"net"

	tls_util "github.com/go-gost/gost/pkg/common/util/tls"
	mdata "github.com/go-gost/gost/pkg/metadata"
)

type metadata struct {
	tlsConfig *tls.Config
}

func (d *http2Dialer) parseMetadata(md mdata.Metadata) (err error) {
	const (
		certFile   = "certFile"
		keyFile    = "keyFile"
		caFile     = "caFile"
		secure     = "secure"
		serverName = "serverName"
	)

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

	return
}
