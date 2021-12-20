package http

import (
	"net/http"
	"strings"

	"github.com/go-gost/gost/pkg/auth"
	mdata "github.com/go-gost/gost/pkg/metadata"
)

type metadata struct {
	retryCount    int
	authenticator auth.Authenticator
	probeResist   *probeResist
	sni           bool
	enableUDP     bool
	header        http.Header
}

func (h *httpHandler) parseMetadata(md mdata.Metadata) error {
	const (
		header         = "header"
		users          = "users"
		probeResistKey = "probeResist"
		knock          = "knock"
		retryCount     = "retry"
		sni            = "sni"
		enableUDP      = "udp"
	)

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

	if m := mdata.GetStringMapString(md, header); len(m) > 0 {
		hd := http.Header{}
		for k, v := range m {
			hd.Add(k, v)
		}
		h.md.header = hd
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
	h.md.retryCount = mdata.GetInt(md, retryCount)
	h.md.sni = mdata.GetBool(md, sni)
	h.md.enableUDP = mdata.GetBool(md, enableUDP)

	return nil
}

type probeResist struct {
	Type  string
	Value string
	Knock string
}
