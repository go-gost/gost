package ss

import (
	"strings"
	"time"

	"github.com/go-gost/gost/pkg/common/util/ss"
	mdata "github.com/go-gost/gost/pkg/metadata"
	"github.com/shadowsocks/go-shadowsocks2/core"
)

type metadata struct {
	cipher      core.Cipher
	readTimeout time.Duration
	retryCount  int
}

func (h *ssHandler) parseMetadata(md mdata.Metadata) (err error) {
	const (
		users       = "users"
		key         = "key"
		readTimeout = "readTimeout"
		retryCount  = "retry"
	)

	var method, password string
	if auths := mdata.GetStrings(md, users); len(auths) > 0 {
		auth := auths[0]
		ss := strings.SplitN(auth, ":", 2)
		if len(ss) == 1 {
			method = ss[0]
		} else {
			method, password = ss[0], ss[1]
		}
	}
	h.md.cipher, err = ss.ShadowCipher(method, password, mdata.GetString(md, key))
	if err != nil {
		return
	}

	h.md.readTimeout = mdata.GetDuration(md, readTimeout)
	h.md.retryCount = mdata.GetInt(md, retryCount)

	return
}
