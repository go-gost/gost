package ss

import (
	"strings"
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
		users       = "users"
		key         = "key"
		readTimeout = "readTimeout"
		retryCount  = "retry"
		bufferSize  = "bufferSize"
	)

	var method, password string
	if v, _ := md.Get(users).([]interface{}); len(v) > 0 {
		for _, auth := range v {
			if s, _ := auth.(string); s != "" {
				ss := strings.SplitN(s, ":", 2)
				if len(ss) == 1 {
					method = ss[0]
				} else {
					method, password = ss[0], ss[1]
				}
			}
		}
	}
	h.md.cipher, err = ss.ShadowCipher(method, password, md.GetString(key))
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
