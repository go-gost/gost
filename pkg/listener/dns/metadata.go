package dns

import (
	"crypto/tls"
	"time"

	tls_util "github.com/go-gost/gost/pkg/common/util/tls"
	mdata "github.com/go-gost/gost/pkg/metadata"
)

const (
	defaultBacklog = 128
)

type metadata struct {
	mode           string
	readBufferSize int
	readTimeout    time.Duration
	writeTimeout   time.Duration
	tlsConfig      *tls.Config
	backlog        int
}

func (l *dnsListener) parseMetadata(md mdata.Metadata) (err error) {
	const (
		mode           = "mode"
		readBufferSize = "readBufferSize"

		certFile = "certFile"
		keyFile  = "keyFile"
		caFile   = "caFile"

		backlog = "backlog"
	)

	l.md.mode = mdata.GetString(md, mode)
	l.md.readBufferSize = mdata.GetInt(md, readBufferSize)

	l.md.tlsConfig, err = tls_util.LoadServerConfig(
		mdata.GetString(md, certFile),
		mdata.GetString(md, keyFile),
		mdata.GetString(md, caFile),
	)
	if err != nil {
		return
	}
	l.md.backlog = mdata.GetInt(md, backlog)
	if l.md.backlog <= 0 {
		l.md.backlog = defaultBacklog
	}

	return
}
