package http

import (
	"net/url"
	"time"
)

const (
	connectTimeout = "timeout"
	userAgent      = "userAgent"
	auth           = "auth"
)

const (
	defaultUserAgent = "Chrome/78.0.3904.106"
)

type metadata struct {
	connectTimeout time.Duration
	UserAgent      string
	User           *url.Userinfo
}
