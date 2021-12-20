package v4

import (
	"time"

	"github.com/go-gost/gost/pkg/auth"
	mdata "github.com/go-gost/gost/pkg/metadata"
)

type metadata struct {
	authenticator auth.Authenticator
	readTimeout   time.Duration
	retryCount    int
}

func (h *socks4Handler) parseMetadata(md mdata.Metadata) (err error) {
	const (
		users       = "users"
		readTimeout = "readTimeout"
		retryCount  = "retry"
	)

	if auths := mdata.GetStrings(md, users); len(auths) > 0 {
		authenticator := auth.NewLocalAuthenticator(nil)
		for _, auth := range auths {
			if auth != "" {
				authenticator.Add(auth, "")
			}
		}
		h.md.authenticator = authenticator
	}

	h.md.readTimeout = mdata.GetDuration(md, readTimeout)
	h.md.retryCount = mdata.GetInt(md, retryCount)
	return
}
