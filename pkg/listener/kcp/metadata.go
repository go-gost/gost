package kcp

const (
	connQueueSize = "connQueueSize"
)

const (
	defaultQueueSize = 128
)

type metadata struct {
	config *Config

	connQueueSize int
}
