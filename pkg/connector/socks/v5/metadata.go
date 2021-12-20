package v5

import (
	"crypto/tls"
	"net/url"
	"strings"
	"time"

	mdata "github.com/go-gost/gost/pkg/metadata"
)

type metadata struct {
	connectTimeout time.Duration
	User           *url.Userinfo
	tlsConfig      *tls.Config
	noTLS          bool
}

func (c *socks5Connector) parseMetadata(md mdata.Metadata) (err error) {
	const (
		connectTimeout = "timeout"
		user           = "user"
		noTLS          = "notls"
	)

	if v := mdata.GetString(md, user); v != "" {
		ss := strings.SplitN(v, ":", 2)
		if len(ss) == 1 {
			c.md.User = url.User(ss[0])
		} else {
			c.md.User = url.UserPassword(ss[0], ss[1])
		}
	}

	c.md.connectTimeout = mdata.GetDuration(md, connectTimeout)
	c.md.noTLS = mdata.GetBool(md, noTLS)

	return
}
