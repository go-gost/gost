package v4

import (
	"net/url"
	"time"

	mdata "github.com/go-gost/gost/pkg/metadata"
)

type metadata struct {
	connectTimeout time.Duration
	User           *url.Userinfo
	disable4a      bool
}

func (c *socks4Connector) parseMetadata(md mdata.Metadata) (err error) {
	const (
		connectTimeout = "timeout"
		user           = "user"
		disable4a      = "disable4a"
	)

	if v := mdata.GetString(md, user); v != "" {
		c.md.User = url.User(v)
	}
	c.md.connectTimeout = mdata.GetDuration(md, connectTimeout)
	c.md.disable4a = mdata.GetBool(md, disable4a)

	return
}
