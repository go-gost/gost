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
	udpBufferSize  int
}

func (c *ssuConnector) parseMetadata(md md.Metadata) (err error) {
	const (
		user           = "user"
		key            = "key"
		connectTimeout = "timeout"
		udpBufferSize  = "udpBufferSize" // udp buffer size
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

	if c.md.udpBufferSize > 0 {
		if c.md.udpBufferSize < 512 {
			c.md.udpBufferSize = 512
		}
		if c.md.udpBufferSize > 65*1024 {
			c.md.udpBufferSize = 65 * 1024
		}
	} else {
		c.md.udpBufferSize = 4096
	}

	return
}
