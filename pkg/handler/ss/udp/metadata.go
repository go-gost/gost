package ss

import (
	"time"

	"github.com/go-gost/gost/pkg/common/util/ss"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/shadowsocks/go-shadowsocks2/core"
)

type metadata struct {
	cipher      core.Cipher
	readTimeout time.Duration
	retryCount  int
	bufferSize  int
}

func (h *ssuHandler) parseMetadata(md md.Metadata) (err error) {
	const (
		method      = "method"
		password    = "password"
		key         = "key"
		readTimeout = "readTimeout"
		retryCount  = "retry"
		bufferSize  = "bufferSize"
	)

	h.md.cipher, err = ss.ShadowCipher(
		md.GetString(method),
		md.GetString(password),
		md.GetString(key),
	)
	if err != nil {
		return
	}

	h.md.readTimeout = md.GetDuration(readTimeout)
	h.md.retryCount = md.GetInt(retryCount)

	h.md.bufferSize = md.GetInt(bufferSize)
	if h.md.bufferSize > 0 {
		if h.md.bufferSize < 512 {
			h.md.bufferSize = 512 // min buffer size
		}
		if h.md.bufferSize > 65*1024 {
			h.md.bufferSize = 65 * 1024 // max buffer size
		}
	} else {
		h.md.bufferSize = 4096 // default buffer size
	}
	return
}
