package udp

import "time"

const (
	dialTimeout = "dialTimeout"
)

const (
	defaultDialTimeout = 5 * time.Second
)

type metadata struct {
	dialTimeout time.Duration
}
