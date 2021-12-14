package http2

import (
	"crypto/tls"
	"net/http"
	"time"

	tls_util "github.com/go-gost/gost/pkg/common/util/tls"
	md "github.com/go-gost/gost/pkg/metadata"
)

const (
	defaultBacklog = 128
)

type metadata struct {
	path              string
	tlsConfig         *tls.Config
	handshakeTimeout  time.Duration
	readHeaderTimeout time.Duration
	readBufferSize    int
	writeBufferSize   int
	enableCompression bool
	responseHeader    http.Header
	backlog           int
}

func (l *http2Listener) parseMetadata(md md.Metadata) (err error) {
	const (
		path              = "path"
		certFile          = "certFile"
		keyFile           = "keyFile"
		caFile            = "caFile"
		handshakeTimeout  = "handshakeTimeout"
		readHeaderTimeout = "readHeaderTimeout"
		readBufferSize    = "readBufferSize"
		writeBufferSize   = "writeBufferSize"
		backlog           = "backlog"
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
	return
}
