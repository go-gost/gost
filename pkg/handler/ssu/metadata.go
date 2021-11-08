package ssu

import (
	"time"

	"github.com/shadowsocks/go-shadowsocks2/core"
)

const (
	method      = "method"
	password    = "password"
	key         = "key"
	readTimeout = "readTimeout"
	retryCount  = "retry"
	bufferSize  = "bufferSize"
)

type metadata struct {
	cipher      core.Cipher
	readTimeout time.Duration
	retryCount  int
	bufferSize  int
}
