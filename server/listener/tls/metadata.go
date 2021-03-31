package tls

import (
	"crypto/tls"
	"time"
)

const (
	addr            = "addr"
	certFile        = "certFile"
	keyFile         = "keyFile"
	caFile          = "caFile"
	keepAlivePeriod = "keepAlivePeriod"
)

type metadata struct {
	addr            string
	tlsConfig       *tls.Config
	keepAlivePeriod time.Duration
}
