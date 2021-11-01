package ss

import (
	"time"

	"github.com/shadowsocks/go-shadowsocks2/core"
)

const (
	method      = "method"
	password    = "password"
	key         = "key"
	readTimeout = "readTimeout"
)

type metadata struct {
	cipher      core.Cipher
	readTimeout time.Duration
}
