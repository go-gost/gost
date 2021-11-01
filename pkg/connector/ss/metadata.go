package ss

import (
	"time"

	"github.com/shadowsocks/go-shadowsocks2/core"
)

const (
	method         = "method"
	password       = "password"
	key            = "key"
	connectTimeout = "timeout"
	noDelay        = "noDelay"
)

type metadata struct {
	cipher         core.Cipher
	connectTimeout time.Duration
	noDelay        bool
}
