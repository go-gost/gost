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
	enableBind    bool
	udpBufferSize int
}

func (h *relayHandler) parseMetadata(md md.Metadata) (err error) {
	const (
		users         = "users"
		readTimeout   = "readTimeout"
		retryCount    = "retry"
		enableBind    = "bind"
		udpBufferSize = "udpBufferSize"
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
	h.md.readTimeout = md.GetDuration(readTimeout)
	h.md.retryCount = md.GetInt(retryCount)
	h.md.enableBind = md.GetBool(enableBind)
	h.md.udpBufferSize = md.GetInt(udpBufferSize)
	if h.md.udpBufferSize > 0 {
		if h.md.udpBufferSize < 512 {
			h.md.udpBufferSize = 512 // min buffer size
		}
		if h.md.udpBufferSize > 65*1024 {
			h.md.udpBufferSize = 65 * 1024 // max buffer size
		}
	} else {
		h.md.udpBufferSize = 1024 // default buffer size
	}
	return
}
