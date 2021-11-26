package redirect

import (
	md "github.com/go-gost/gost/pkg/metadata"
)

type metadata struct {
	retryCount int
}

func (h *redirectHandler) parseMetadata(md md.Metadata) (err error) {
	const (
		retryCount = "retry"
	)

	h.md.retryCount = md.GetInt(retryCount)
	return
}
