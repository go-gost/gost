package http2

import (
	"strings"

	"github.com/go-gost/gost/pkg/auth"
	mdata "github.com/go-gost/gost/pkg/metadata"
)

type metadata struct {
	authenticator auth.Authenticator
	proxyAgent    string
	probeResist   *probeResist
	sni           bool
	enableUDP     bool
}

func (h *http2Handler) parseMetadata(md mdata.Metadata) error {
	const (
		proxyAgent     = "proxyAgent"
		users          = "users"
		probeResistKey = "probeResist"
		knock          = "knock"
		sni            = "sni"
		enableUDP      = "udp"
	)

	h.md.proxyAgent = mdata.GetString(md, proxyAgent)

	if auths := mdata.GetStrings(md, users); len(auths) > 0 {
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

	if v := mdata.GetString(md, probeResistKey); v != "" {
		if ss := strings.SplitN(v, ":", 2); len(ss) == 2 {
			h.md.probeResist = &probeResist{
				Type:  ss[0],
				Value: ss[1],
				Knock: mdata.GetString(md, knock),
			}
		}
	}
	h.md.sni = mdata.GetBool(md, sni)
	h.md.enableUDP = mdata.GetBool(md, enableUDP)

	return nil
}

type probeResist struct {
	Type  string
	Value string
	Knock string
}
