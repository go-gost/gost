package v4

import (
	"time"

	"github.com/go-gost/gost/pkg/auth"
	md "github.com/go-gost/gost/pkg/metadata"
)

type metadata struct {
	authenticator auth.Authenticator
	readTimeout   time.Duration
	retryCount    int
}

func (h *socks4Handler) parseMetadata(md md.Metadata) (err error) {
	const (
		authsKey    = "auths"
		readTimeout = "readTimeout"
		retryCount  = "retry"
	)

	if v, _ := md.Get(authsKey).([]interface{}); len(v) > 0 {
		authenticator := auth.NewLocalAuthenticator(nil)
		for _, auth := range v {
			if v, _ := auth.(string); v != "" {
				authenticator.Add(v, "")
			}
		}
		h.md.authenticator = authenticator
	}

	h.md.readTimeout = md.GetDuration(readTimeout)
	h.md.retryCount = md.GetInt(retryCount)
	return
}
