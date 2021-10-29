package http

import "github.com/go-gost/gost/pkg/auth"

const (
	addr       = "addr"
	proxyAgent = "proxyAgent"
	auths      = "auths"
)

type metadata struct {
	addr          string
	authenticator auth.Authenticator
	proxyAgent    string
	retryCount    int
}
