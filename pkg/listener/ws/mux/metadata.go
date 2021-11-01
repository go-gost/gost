package mux

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

	muxKeepAliveDisabled = "muxKeepAliveDisabled"
	muxKeepAlivePeriod   = "muxKeepAlivePeriod"
	muxKeepAliveTimeout  = "muxKeepAliveTimeout"
	muxMaxFrameSize      = "muxMaxFrameSize"
	muxMaxReceiveBuffer  = "muxMaxReceiveBuffer"
	muxMaxStreamBuffer   = "muxMaxStreamBuffer"
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

	muxKeepAliveDisabled bool
	muxKeepAlivePeriod   time.Duration
	muxKeepAliveTimeout  time.Duration
	muxMaxFrameSize      int
	muxMaxReceiveBuffer  int
	muxMaxStreamBuffer   int
	connQueueSize        int
}
