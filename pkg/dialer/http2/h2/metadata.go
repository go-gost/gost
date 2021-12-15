package h2

import (
	"crypto/tls"
	"net"

	tls_util "github.com/go-gost/gost/pkg/common/util/tls"
	md "github.com/go-gost/gost/pkg/metadata"
)

type metadata struct {
	path      string
	host      string
	tlsConfig *tls.Config
}

func (d *h2Dialer) parseMetadata(md md.Metadata) (err error) {
	const (
		certFile   = "certFile"
		keyFile    = "keyFile"
		caFile     = "caFile"
		secure     = "secure"
		serverName = "serverName"
		path       = "path"
	)

	d.md.host = md.GetString(serverName)
	sn, _, _ := net.SplitHostPort(d.md.host)
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

	d.md.path = md.GetString(path)

	return
}
