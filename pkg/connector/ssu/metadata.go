package ssu

import (
	"time"

	"github.com/shadowsocks/go-shadowsocks2/core"
)

const (
	method         = "method"
	password       = "password"
	key            = "key"
	connectTimeout = "timeout"
	bufferSize     = "bufferSize"
)

type metadata struct {
	cipher         core.Cipher
	connectTimeout time.Duration
	bufferSize     int
}
