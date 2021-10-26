package kcp

const (
	addr = "addr"

	connQueueSize = "connQueueSize"
)

const (
	defaultQueueSize = 128
)

type metadata struct {
	addr   string
	config *Config

	connQueueSize int
}
