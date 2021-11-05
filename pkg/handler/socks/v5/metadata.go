package v5

import (
	"crypto/tls"
	"strings"
	"time"

	"github.com/go-gost/gost/pkg/auth"
	"github.com/go-gost/gost/pkg/internal/utils"
	md "github.com/go-gost/gost/pkg/metadata"
)

const (
	certFile    = "certFile"
	keyFile     = "keyFile"
	caFile      = "caFile"
	authsKey    = "auths"
	readTimeout = "readTimeout"
	retryCount  = "retry"
	noTLS       = "notls"
)

type metadata struct {
	tlsConfig     *tls.Config
	authenticator auth.Authenticator
	readTimeout   time.Duration
	retryCount    int
	noTLS         bool
}

func (h *socks5Handler) parseMetadata(md md.Metadata) error {
	var err error
	h.md.tlsConfig, err = utils.LoadTLSConfig(
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
	h.md.retryCount = md.GetInt(retryCount)
	h.md.noTLS = md.GetBool(noTLS)

	return nil
}
