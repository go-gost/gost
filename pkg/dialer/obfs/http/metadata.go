package http

import (
	"fmt"

	md "github.com/go-gost/gost/pkg/metadata"
)

type metadata struct {
	host    string
	headers map[string]string
}

func (d *obfsHTTPDialer) parseMetadata(md md.Metadata) (err error) {
	const (
		headers = "headers"
		host    = "host"
	)

	if mm, _ := md.Get(headers).(map[interface{}]interface{}); len(mm) > 0 {
		m := make(map[string]string)
		for k, v := range mm {
			m[fmt.Sprintf("%v", k)] = fmt.Sprintf("%v", v)
		}
		d.md.headers = m
	}
	d.md.host = md.GetString(host)
	return
}
