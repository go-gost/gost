package ss

import (
	"math"
	"time"

	mdata "github.com/go-gost/gost/pkg/metadata"
)

type metadata struct {
	key            string
	connectTimeout time.Duration
	bufferSize     int
}

func (c *ssuConnector) parseMetadata(md mdata.Metadata) (err error) {
	const (
		key            = "key"
		connectTimeout = "timeout"
		bufferSize     = "bufferSize" // udp buffer size
	)

	c.md.key = mdata.GetString(md, key)
	c.md.connectTimeout = mdata.GetDuration(md, connectTimeout)

	if bs := mdata.GetInt(md, bufferSize); bs > 0 {
		c.md.bufferSize = int(math.Min(math.Max(float64(bs), 512), 64*1024))
	} else {
		c.md.bufferSize = 1024
	}

	return
}
