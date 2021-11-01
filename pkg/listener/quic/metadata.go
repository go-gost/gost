package quic

import (
	"crypto/tls"
	"time"
)

const (
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
	tlsConfig        *tls.Config
	keepAlive        bool
	HandshakeTimeout time.Duration
	MaxIdleTimeout   time.Duration

	cipherKey     []byte
	connQueueSize int
}
