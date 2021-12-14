package h2

import (
	"crypto/tls"
	"net/http"
	"time"

	tls_util "github.com/go-gost/gost/pkg/common/util/tls"
	md "github.com/go-gost/gost/pkg/metadata"
)

const (
	defaultQueueSize = 128
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
	connQueueSize     int
	keepAlivePeriod   time.Duration
}

func (l *h2Listener) parseMetadata(md md.Metadata) (err error) {
	const (
		path              = "path"
		certFile          = "certFile"
		keyFile           = "keyFile"
		caFile            = "caFile"
		handshakeTimeout  = "handshakeTimeout"
		readHeaderTimeout = "readHeaderTimeout"
		readBufferSize    = "readBufferSize"
		writeBufferSize   = "writeBufferSize"
		connQueueSize     = "connQueueSize"
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
