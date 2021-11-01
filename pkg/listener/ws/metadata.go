package ws

import (
	"crypto/tls"
	"net/http"
	"time"
)

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
