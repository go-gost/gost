package v4

import (
	"net/url"
	"time"
)

const (
	connectTimeout = "timeout"
	auth           = "auth"
	disable4a      = "disable4a"
)

type metadata struct {
	connectTimeout time.Duration
	User           *url.Userinfo
	disable4a      bool
}
