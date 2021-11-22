package ss

import (
	"strings"
	"time"

	"github.com/go-gost/gost/pkg/common/util/ss"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/shadowsocks/go-shadowsocks2/core"
)

type metadata struct {
	cipher         core.Cipher
	connectTimeout time.Duration
	noDelay        bool
}

func (c *ssConnector) parseMetadata(md md.Metadata) (err error) {
	const (
		user           = "user"
		key            = "key"
		connectTimeout = "timeout"
		noDelay        = "nodelay"
	)

	var method, password string
	if v := md.GetString(user); v != "" {
		ss := strings.SplitN(v, ":", 2)
		if len(ss) == 1 {
			method = ss[0]
		} else {
			method, password = ss[0], ss[1]
		}
	}
	c.md.cipher, err = ss.ShadowCipher(method, password, md.GetString(key))
	if err != nil {
		return
	}

	c.md.connectTimeout = md.GetDuration(connectTimeout)
	c.md.noDelay = md.GetBool(noDelay)

	return
}
