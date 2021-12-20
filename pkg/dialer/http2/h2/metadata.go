package h2

import (
	"crypto/tls"
	"net"

	tls_util "github.com/go-gost/gost/pkg/common/util/tls"
	mdata "github.com/go-gost/gost/pkg/metadata"
)

type metadata struct {
	path      string
	host      string
	tlsConfig *tls.Config
}

func (d *h2Dialer) parseMetadata(md mdata.Metadata) (err error) {
	const (
		certFile   = "certFile"
		keyFile    = "keyFile"
		caFile     = "caFile"
		secure     = "secure"
		serverName = "serverName"
		path       = "path"
	)

	d.md.host = mdata.GetString(md, serverName)
	sn, _, _ := net.SplitHostPort(d.md.host)
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

	d.md.path = mdata.GetString(md, path)

	return
}
