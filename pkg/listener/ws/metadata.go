package ws

import (
	"crypto/tls"
	"net/http"
	"time"

	tls_util "github.com/go-gost/gost/pkg/common/util/tls"
	md "github.com/go-gost/gost/pkg/metadata"
)

const (
	defaultPath      = "/ws"
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
}

func (l *wsListener) parseMetadata(md md.Metadata) (err error) {
	const (
		path              = "path"
		certFile          = "certFile"
		keyFile           = "keyFile"
		caFile            = "caFile"
		handshakeTimeout  = "handshakeTimeout"
		readHeaderTimeout = "readHeaderTimeout"
		readBufferSize    = "readBufferSize"
		writeBufferSize   = "writeBufferSize"
		enableCompression = "enableCompression"
		responseHeader    = "responseHeader"
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

	l.md.path = md.GetString(path)
	l.md.connQueueSize = md.GetInt(connQueueSize)
	if l.md.connQueueSize <= 0 {
		l.md.connQueueSize = defaultQueueSize
	}
	l.md.enableCompression = md.GetBool(enableCompression)
	l.md.readBufferSize = md.GetInt(readBufferSize)
	l.md.writeBufferSize = md.GetInt(writeBufferSize)
	l.md.handshakeTimeout = md.GetDuration(handshakeTimeout)
	l.md.readHeaderTimeout = md.GetDuration(readHeaderTimeout)

	return
}
