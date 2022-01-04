package ss

import (
	"time"

	mdata "github.com/go-gost/gost/pkg/metadata"
)

type metadata struct {
	key         string
	readTimeout time.Duration
}

func (h *ssHandler) parseMetadata(md mdata.Metadata) (err error) {
	const (
		key         = "key"
		readTimeout = "readTimeout"
	)

	h.md.key = mdata.GetString(md, key)
	h.md.readTimeout = mdata.GetDuration(md, readTimeout)

	return
}
