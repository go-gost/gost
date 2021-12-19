package http2

import (
	"strings"

	"github.com/go-gost/gost/pkg/auth"
	md "github.com/go-gost/gost/pkg/metadata"
)

type metadata struct {
	authenticator auth.Authenticator
	proxyAgent    string
	retryCount    int
	probeResist   *probeResist
	sni           bool
	enableUDP     bool
}

func (h *http2Handler) parseMetadata(md md.Metadata) error {
	const (
		proxyAgent     = "proxyAgent"
		users          = "users"
		probeResistKey = "probeResist"
		knock          = "knock"
		retryCount     = "retry"
		sni            = "sni"
		enableUDP      = "udp"
	)

	h.md.proxyAgent = md.GetString(proxyAgent)

	if auths := md.GetStrings(users); len(auths) > 0 {
		authenticator := auth.NewLocalAuthenticator(nil)
		for _, auth := range auths {
			ss := strings.SplitN(auth, ":", 2)
			if len(ss) == 1 {
				authenticator.Add(ss[0], "")
			} else {
				authenticator.Add(ss[0], ss[1])
			}
		}
		h.md.authenticator = authenticator
	}

	if v := md.GetString(probeResistKey); v != "" {
		if ss := strings.SplitN(v, ":", 2); len(ss) == 2 {
			h.md.probeResist = &probeResist{
				Type:  ss[0],
				Value: ss[1],
				Knock: md.GetString(knock),
			}
		}
	}
	h.md.retryCount = md.GetInt(retryCount)
	h.md.sni = md.GetBool(sni)
	h.md.enableUDP = md.GetBool(enableUDP)

	return nil
}

type probeResist struct {
	Type  string
	Value string
	Knock string
}
