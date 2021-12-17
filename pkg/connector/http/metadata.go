package http

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	md "github.com/go-gost/gost/pkg/metadata"
)

type metadata struct {
	connectTimeout time.Duration
	User           *url.Userinfo
	header         http.Header
}

func (c *httpConnector) parseMetadata(md md.Metadata) (err error) {
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

	if mm, _ := md.Get(header).(map[interface{}]interface{}); len(mm) > 0 {
		h := http.Header{}
		for k, v := range mm {
			h.Add(fmt.Sprintf("%v", k), fmt.Sprintf("%v", v))
		}
		c.md.header = h
	}

	return
}
