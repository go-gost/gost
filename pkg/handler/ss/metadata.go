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
}

func (h *ssHandler) parseMetadata(md md.Metadata) (err error) {
	const (
		users       = "users"
		key         = "key"
		readTimeout = "readTimeout"
		retryCount  = "retry"
	)

	var method, password string
	if v, _ := md.Get(users).([]interface{}); len(v) > 0 {
		h.logger.Info(v)
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

	return
}
