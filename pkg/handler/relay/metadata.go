package relay

import (
	"math"
	"strings"
	"time"

	"github.com/go-gost/gost/pkg/auth"
	mdata "github.com/go-gost/gost/pkg/metadata"
)

type metadata struct {
	authenticator auth.Authenticator
	readTimeout   time.Duration
	enableBind    bool
	udpBufferSize int
	noDelay       bool
}

func (h *relayHandler) parseMetadata(md mdata.Metadata) (err error) {
	const (
		users         = "users"
		readTimeout   = "readTimeout"
		enableBind    = "bind"
		udpBufferSize = "udpBufferSize"
		noDelay       = "nodelay"
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

	h.md.readTimeout = mdata.GetDuration(md, readTimeout)
	h.md.enableBind = mdata.GetBool(md, enableBind)
	h.md.noDelay = mdata.GetBool(md, noDelay)

	if bs := mdata.GetInt(md, udpBufferSize); bs > 0 {
		h.md.udpBufferSize = int(math.Min(math.Max(float64(bs), 512), 64*1024))
	} else {
		h.md.udpBufferSize = 1024
	}
	return
}
