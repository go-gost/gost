package http

import (
	"fmt"
	"net/http"

	md "github.com/go-gost/gost/pkg/metadata"
)

type metadata struct {
	host   string
	header http.Header
}

func (d *obfsHTTPDialer) parseMetadata(md md.Metadata) (err error) {
	const (
		header = "header"
		host   = "host"
	)

	if mm, _ := md.Get(header).(map[interface{}]interface{}); len(mm) > 0 {
		h := http.Header{}
		for k, v := range mm {
			h.Add(fmt.Sprintf("%v", k), fmt.Sprintf("%v", v))
		}
		d.md.header = h
	}
	d.md.host = md.GetString(host)
	return
}
