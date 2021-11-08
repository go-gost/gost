package udp

import "time"

const (
	defaultTTL            = 60 * time.Second
	defaultReadBufferSize = 1024
	defaultReadQueueSize  = 128
	defaultConnQueueSize  = 128
)

const (
	ttl            = "ttl"
	readBufferSize = "readBufferSize"
	readQueueSize  = "readQueueSize"
	connQueueSize  = "connQueueSize"
)

type metadata struct {
	ttl time.Duration

	readBufferSize int
	readQueueSize  int
	connQueueSize  int
}
