package http

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-gost/gost/pkg/auth"
	md "github.com/go-gost/gost/pkg/metadata"
)

type metadata struct {
	retryCount    int
	authenticator auth.Authenticator
	probeResist   *probeResist
	sni           bool
	enableUDP     bool
	header        http.Header
}

func (h *httpHandler) parseMetadata(md md.Metadata) error {
	const (
		header         = "header"
		users          = "users"
		probeResistKey = "probeResist"
		knock          = "knock"
		retryCount     = "retry"
		sni            = "sni"
		enableUDP      = "udp"
	)

	if v, _ := md.Get(users).([]interface{}); len(v) > 0 {
		authenticator := auth.NewLocalAuthenticator(nil)
		for _, auth := range v {
			if s, _ := auth.(string); s != "" {
				ss := strings.SplitN(s, ":", 2)
				if len(ss) == 1 {
					authenticator.Add(ss[0], "")
				} else {
					authenticator.Add(ss[0], ss[1])
				}
			}
		}
		h.md.authenticator = authenticator
	}

	if mm, _ := md.Get(header).(map[interface{}]interface{}); len(mm) > 0 {
		hd := http.Header{}
		for k, v := range mm {
			hd.Add(fmt.Sprintf("%v", k), fmt.Sprintf("%v", v))
		}
		h.md.header = hd
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
