package tls

import "time"

const (
	addr            = "addr"
	keepAlive       = "keepAlive"
	keepAlivePeriod = "keepAlivePeriod"
)

const (
	defaultKeepAlivePeriod = 180 * time.Second
)

type metadata struct {
	addr            string
	keepAlive       bool
	keepAlivePeriod time.Duration
}
