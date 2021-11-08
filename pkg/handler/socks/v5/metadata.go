package v5

import (
	"crypto/tls"
	"strings"
	"time"

	"github.com/go-gost/gost/pkg/auth"
	util_tls "github.com/go-gost/gost/pkg/internal/utils/tls"
	md "github.com/go-gost/gost/pkg/metadata"
)

const (
	certFile      = "certFile"
	keyFile       = "keyFile"
	caFile        = "caFile"
	authsKey      = "auths"
	readTimeout   = "readTimeout"
	timeout       = "timeout"
	retryCount    = "retry"
	noTLS         = "notls"
	udpBufferSize = "udpBufferSize"
)

type metadata struct {
	tlsConfig     *tls.Config
	authenticator auth.Authenticator
	timeout       time.Duration
	readTimeout   time.Duration
	retryCount    int
	noTLS         bool
	udpBufferSize int
}

func (h *socks5Handler) parseMetadata(md md.Metadata) error {
	var err error
	h.md.tlsConfig, err = util_tls.LoadTLSConfig(
		md.GetString(certFile),
		md.GetString(keyFile),
		md.GetString(caFile),
	)
	if err != nil {
		h.logger.Warn("parse tls config: ", err)
	}

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
	h.md.timeout = md.GetDuration(timeout)
	h.md.retryCount = md.GetInt(retryCount)
	h.md.noTLS = md.GetBool(noTLS)

	h.md.udpBufferSize = md.GetInt(udpBufferSize)
	if h.md.udpBufferSize > 0 {
		if h.md.udpBufferSize < 512 {
			h.md.udpBufferSize = 512 // min buffer size
		}
		if h.md.udpBufferSize > 65*1024 {
			h.md.udpBufferSize = 65 * 1024 // max buffer size
		}
	} else {
		h.md.udpBufferSize = 4096 // default buffer size
	}

	return nil
}
