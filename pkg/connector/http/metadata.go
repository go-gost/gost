package http

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	mdata "github.com/go-gost/gost/pkg/metadata"
)

type metadata struct {
	connectTimeout time.Duration
	User           *url.Userinfo
	header         http.Header
}

func (c *httpConnector) parseMetadata(md mdata.Metadata) (err error) {
	const (
		connectTimeout = "timeout"
		user           = "user"
		header         = "header"
	)

	c.md.connectTimeout = md.GetDuration(connectTimeout)

	if v := md.GetString(user); v != "" {
		ss := strings.SplitN(v, ":", 2)
		if len(ss) == 1 {
			c.md.User = url.User(ss[0])
		} else {
			c.md.User = url.UserPassword(ss[0], ss[1])
		}
	}

	if mm := mdata.GetStringMapString(md, header); len(mm) > 0 {
		hd := http.Header{}
		for k, v := range mm {
			hd.Add(k, v)
		}
		c.md.header = hd
	}

	return
}
