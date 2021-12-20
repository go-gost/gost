package local

import (
	"time"

	mdata "github.com/go-gost/gost/pkg/metadata"
)

type metadata struct {
	readTimeout time.Duration
	retryCount  int
}

func (h *forwardHandler) parseMetadata(md mdata.Metadata) (err error) {
	const (
		readTimeout = "readTimeout"
		retryCount  = "retry"
	)

	h.md.readTimeout = mdata.GetDuration(md, readTimeout)
	h.md.retryCount = mdata.GetInt(md, retryCount)
	return
}
