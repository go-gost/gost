package v4

import (
	"net/url"
	"time"

	md "github.com/go-gost/gost/pkg/metadata"
)

type metadata struct {
	connectTimeout time.Duration
	User           *url.Userinfo
	disable4a      bool
}

func (c *socks4Connector) parseMetadata(md md.Metadata) (err error) {
	const (
		connectTimeout = "timeout"
		auth           = "auth"
		disable4a      = "disable4a"
	)

	if v := md.GetString(auth); v != "" {
		c.md.User = url.User(v)
	}
	c.md.connectTimeout = md.GetDuration(connectTimeout)
	c.md.disable4a = md.GetBool(disable4a)

	return
}
