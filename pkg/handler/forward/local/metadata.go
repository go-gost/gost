package local

import (
	"time"
)

const (
	readTimeout = "readTimeout"
	retryCount  = "retry"
)

type metadata struct {
	readTimeout time.Duration
	retryCount  int
}
