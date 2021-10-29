package http

import "time"

const (
	keepAlive       = "keepAlive"
	keepAlivePeriod = "keepAlivePeriod"
)

const (
	defaultKeepAlivePeriod = 180 * time.Second
)

type metadata struct {
	keepAlive       bool
	keepAlivePeriod time.Duration
}
