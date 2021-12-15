package h2

import (
	"crypto/tls"

	tls_util "github.com/go-gost/gost/pkg/common/util/tls"
	md "github.com/go-gost/gost/pkg/metadata"
)

const (
	defaultBacklog = 128
)

type metadata struct {
	path      string
	tlsConfig *tls.Config
	backlog   int
}

func (l *h2Listener) parseMetadata(md md.Metadata) (err error) {
	const (
		path     = "path"
		certFile = "certFile"
		keyFile  = "keyFile"
		caFile   = "caFile"
		backlog  = "backlog"
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

	l.md.path = md.GetString(path)
	return
}
