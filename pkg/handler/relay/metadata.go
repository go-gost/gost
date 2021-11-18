package relay

import (
	"strings"
	"time"

	"github.com/go-gost/gost/pkg/auth"
	md "github.com/go-gost/gost/pkg/metadata"
)

type metadata struct {
	authenticator auth.Authenticator
	readTimeout   time.Duration
	retryCount    int
}

func (h *relayHandler) parseMetadata(md md.Metadata) (err error) {
	const (
		authsKey    = "auths"
		readTimeout = "readTimeout"
		retryCount  = "retry"
	)

	if v, _ := md.Get(authsKey).([]interface{}); len(v) > 0 {
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
	h.md.readTimeout = md.GetDuration(readTimeout)
	h.md.retryCount = md.GetInt(retryCount)
	return
}
