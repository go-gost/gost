package redirect

import (
	mdata "github.com/go-gost/gost/pkg/metadata"
)

type metadata struct {
	retryCount int
}

func (h *redirectHandler) parseMetadata(md mdata.Metadata) (err error) {
	const (
		retryCount = "retry"
	)

	h.md.retryCount = mdata.GetInt(md, retryCount)
	return
}
