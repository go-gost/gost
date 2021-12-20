package v5

import (
	"crypto/tls"
	"math"
	"strings"
	"time"

	"github.com/go-gost/gost/pkg/auth"
	tls_util "github.com/go-gost/gost/pkg/common/util/tls"
	mdata "github.com/go-gost/gost/pkg/metadata"
)

type metadata struct {
	tlsConfig         *tls.Config
	authenticator     auth.Authenticator
	timeout           time.Duration
	readTimeout       time.Duration
	retryCount        int
	noTLS             bool
	enableBind        bool
	enableUDP         bool
	udpBufferSize     int
	compatibilityMode bool
}

func (h *socks5Handler) parseMetadata(md mdata.Metadata) (err error) {
	const (
		certFile          = "certFile"
		keyFile           = "keyFile"
		caFile            = "caFile"
		users             = "users"
		readTimeout       = "readTimeout"
		timeout           = "timeout"
		retryCount        = "retry"
		noTLS             = "notls"
		enableBind        = "bind"
		enableUDP         = "udp"
		udpBufferSize     = "udpBufferSize"
		compatibilityMode = "comp"
	)

	h.md.tlsConfig, err = tls_util.LoadServerConfig(
		mdata.GetString(md, certFile),
		mdata.GetString(md, keyFile),
		mdata.GetString(md, caFile),
	)
	if err != nil {
		return
	}

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
	h.md.timeout = mdata.GetDuration(md, timeout)
	h.md.retryCount = mdata.GetInt(md, retryCount)
	h.md.noTLS = mdata.GetBool(md, noTLS)
	h.md.enableBind = mdata.GetBool(md, enableBind)
	h.md.enableUDP = mdata.GetBool(md, enableUDP)

	if bs := mdata.GetInt(md, udpBufferSize); bs > 0 {
		h.md.udpBufferSize = int(math.Min(math.Max(float64(bs), 512), 64*1024))
	} else {
		h.md.udpBufferSize = 1024
	}

	h.md.compatibilityMode = mdata.GetBool(md, compatibilityMode)

	return nil
}
