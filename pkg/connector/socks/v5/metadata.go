package v5

import (
	"crypto/tls"
	"net/url"
	"strings"
	"time"

	md "github.com/go-gost/gost/pkg/metadata"
)

type metadata struct {
	connectTimeout time.Duration
	User           *url.Userinfo
	tlsConfig      *tls.Config
	noTLS          bool
}

func (c *socks5Connector) parseMetadata(md md.Metadata) (err error) {
	const (
		connectTimeout = "timeout"
		auth           = "auth"
		noTLS          = "notls"
	)

	if v := md.GetString(auth); v != "" {
		ss := strings.SplitN(v, ":", 2)
		if len(ss) == 1 {
			c.md.User = url.User(ss[0])
		} else {
			c.md.User = url.UserPassword(ss[0], ss[1])
		}
	}

	c.md.connectTimeout = md.GetDuration(connectTimeout)
	c.md.noTLS = md.GetBool(noTLS)

	return
}
