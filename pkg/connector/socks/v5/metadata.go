package v5

import (
	"crypto/tls"
	"net/url"
	"time"
)

const (
	connectTimeout = "timeout"
	auth           = "auth"
	noTLS          = "notls"
)

type metadata struct {
	connectTimeout time.Duration
	User           *url.Userinfo
	tlsConfig      *tls.Config
	noTLS          bool
}
