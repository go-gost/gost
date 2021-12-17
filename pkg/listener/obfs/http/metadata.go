package http

import (
	"fmt"
	"net/http"

	md "github.com/go-gost/gost/pkg/metadata"
)

type metadata struct {
	header http.Header
}

func (l *obfsListener) parseMetadata(md md.Metadata) (err error) {
	const (
		header = "header"
	)

	if mm, _ := md.Get(header).(map[interface{}]interface{}); len(mm) > 0 {
		h := http.Header{}
		for k, v := range mm {
			h.Add(fmt.Sprintf("%v", k), fmt.Sprintf("%v", v))
		}
		l.md.header = h
	}
	return
}
