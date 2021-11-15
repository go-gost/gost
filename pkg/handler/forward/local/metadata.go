package local

import (
	"time"

	md "github.com/go-gost/gost/pkg/metadata"
)

type metadata struct {
	readTimeout time.Duration
	retryCount  int
}

func (h *localForwardHandler) parseMetadata(md md.Metadata) (err error) {
	const (
		readTimeout = "readTimeout"
		retryCount  = "retry"
	)

	h.md.readTimeout = md.GetDuration(readTimeout)
	h.md.retryCount = md.GetInt(retryCount)
	return
}
