package relay

import (
	"net/url"
	"strings"
	"time"

	md "github.com/go-gost/gost/pkg/metadata"
)

type metadata struct {
	connectTimeout time.Duration
	user           *url.Userinfo
	nodelay        bool
}

func (c *relayConnector) parseMetadata(md md.Metadata) (err error) {
	const (
		user           = "user"
		connectTimeout = "connectTimeout"
		nodelay        = "nodelay"
	)

	if v := md.GetString(user); v != "" {
		ss := strings.SplitN(v, ":", 2)
		if len(ss) == 1 {
			c.md.user = url.User(ss[0])
		} else {
			c.md.user = url.UserPassword(ss[0], ss[1])
		}
	}
	c.md.connectTimeout = md.GetDuration(connectTimeout)
	c.md.nodelay = md.GetBool(nodelay)

	return
}
