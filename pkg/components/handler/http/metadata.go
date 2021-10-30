package http

import "github.com/go-gost/gost/pkg/auth"

const (
	addrKey        = "addr"
	proxyAgentKey  = "proxyAgent"
	authsKey       = "auths"
	probeResistKey = "probeResist"
	knockKey       = "knock"
)

type metadata struct {
	addr          string
	authenticator auth.Authenticator
	proxyAgent    string
	retryCount    int
	probeResist   *probeResist
}

type probeResist struct {
	Type  string
	Value string
	Knock string
}
