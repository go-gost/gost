package ss

import (
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
		method         = "method"
		password       = "password"
		key            = "key"
		connectTimeout = "timeout"
		udpBufferSize  = "udpBufferSize" // udp buffer size
	)

	c.md.cipher, err = ss.ShadowCipher(
		md.GetString(method),
		md.GetString(password),
		md.GetString(key),
	)
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
