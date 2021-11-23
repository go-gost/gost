package v5

import (
	"crypto/tls"
	"strings"
	"time"

	"github.com/go-gost/gost/pkg/auth"
	util_tls "github.com/go-gost/gost/pkg/common/util/tls"
	md "github.com/go-gost/gost/pkg/metadata"
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

func (h *socks5Handler) parseMetadata(md md.Metadata) error {
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

	var err error
	h.md.tlsConfig, err = util_tls.LoadTLSConfig(
		md.GetString(certFile),
		md.GetString(keyFile),
		md.GetString(caFile),
	)
	if err != nil {
		h.logger.Warn("parse tls config: ", err)
	}

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
	h.md.timeout = md.GetDuration(timeout)
	h.md.retryCount = md.GetInt(retryCount)
	h.md.noTLS = md.GetBool(noTLS)
	h.md.enableBind = md.GetBool(enableBind)
	h.md.enableUDP = md.GetBool(enableUDP)

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

	h.md.compatibilityMode = md.GetBool(compatibilityMode)

	return nil
}
