package tun

import (
	mdata "github.com/go-gost/gost/pkg/metadata"
)

type metadata struct {
	key        string
	bufferSize int
}

func (h *tunHandler) parseMetadata(md mdata.Metadata) (err error) {
	const (
		key        = "key"
		bufferSize = "bufferSize"
	)

	h.md.key = mdata.GetString(md, key)
	h.md.bufferSize = mdata.GetInt(md, bufferSize)
	if h.md.bufferSize <= 0 {
		h.md.bufferSize = 1024
	}
	return
}
