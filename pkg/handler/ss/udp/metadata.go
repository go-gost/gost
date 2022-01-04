package ss

import (
	"math"
	"time"

	mdata "github.com/go-gost/gost/pkg/metadata"
)

type metadata struct {
	key         string
	readTimeout time.Duration
	bufferSize  int
}

func (h *ssuHandler) parseMetadata(md mdata.Metadata) (err error) {
	const (
		key         = "key"
		readTimeout = "readTimeout"
		bufferSize  = "bufferSize"
	)

	h.md.key = mdata.GetString(md, key)
	h.md.readTimeout = mdata.GetDuration(md, readTimeout)

	if bs := mdata.GetInt(md, bufferSize); bs > 0 {
		h.md.bufferSize = int(math.Min(math.Max(float64(bs), 512), 64*1024))
	} else {
		h.md.bufferSize = 1024
	}
	return
}
