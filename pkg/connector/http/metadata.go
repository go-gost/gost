package http

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	md "github.com/go-gost/gost/pkg/metadata"
)

type metadata struct {
	connectTimeout time.Duration
	User           *url.Userinfo
	headers        map[string]string
}

func (c *httpConnector) parseMetadata(md md.Metadata) (err error) {
	const (
		connectTimeout = "timeout"
		user           = "user"
		headers        = "headers"
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

	if mm, _ := md.Get(headers).(map[interface{}]interface{}); len(mm) > 0 {
		m := make(map[string]string)
		for k, v := range mm {
			m[fmt.Sprintf("%v", k)] = fmt.Sprintf("%v", v)
		}
		c.md.headers = m
	}

	return
}
