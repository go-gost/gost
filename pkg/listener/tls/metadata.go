package tls

import (
	"crypto/tls"
	"time"
)

const (
	certFile        = "certFile"
	keyFile         = "keyFile"
	caFile          = "caFile"
	keepAlivePeriod = "keepAlivePeriod"
)

type metadata struct {
	tlsConfig       *tls.Config
	keepAlivePeriod time.Duration
}
