package ss

import (
	"strings"
	"time"

	"github.com/go-gost/gost/pkg/common/util/ss"
	mdata "github.com/go-gost/gost/pkg/metadata"
	"github.com/shadowsocks/go-shadowsocks2/core"
)

type metadata struct {
	cipher         core.Cipher
	connectTimeout time.Duration
	noDelay        bool
}

func (c *ssConnector) parseMetadata(md mdata.Metadata) (err error) {
	const (
		user           = "user"
		key            = "key"
		connectTimeout = "timeout"
		noDelay        = "nodelay"
	)

	var method, password string
	if v := mdata.GetString(md, user); v != "" {
		ss := strings.SplitN(v, ":", 2)
		if len(ss) == 1 {
			method = ss[0]
		} else {
			method, password = ss[0], ss[1]
		}
	}
	c.md.cipher, err = ss.ShadowCipher(method, password, mdata.GetString(md, key))
	if err != nil {
		return
	}

	c.md.connectTimeout = mdata.GetDuration(md, connectTimeout)
	c.md.noDelay = mdata.GetBool(md, noDelay)

	return
}
