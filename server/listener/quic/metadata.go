package quic

import (
	"crypto/tls"
	"time"
)

const (
	addr = "addr"

	certFile = "certFile"
	keyFile  = "keyFile"
	caFile   = "caFile"

	keepAlive       = "keepAlive"
	keepAlivePeriod = "keepAlivePeriod"
)

const (
	defaultKeepAlivePeriod = 180 * time.Second
)

type metadata struct {
	addr             string
	tlsConfig        *tls.Config
	keepAlive        bool
	HandshakeTimeout time.Duration
	MaxIdleTimeout   time.Duration

	cipherKey     []byte
	connQueueSize int
}
