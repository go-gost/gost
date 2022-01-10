package v5

import (
	"math"
	"time"

	mdata "github.com/go-gost/gost/pkg/metadata"
)

type metadata struct {
	readTimeout       time.Duration
	noTLS             bool
	enableBind        bool
	enableUDP         bool
	udpBufferSize     int
	compatibilityMode bool
}

func (h *socks5Handler) parseMetadata(md mdata.Metadata) (err error) {
	const (
		readTimeout       = "readTimeout"
		noTLS             = "notls"
		enableBind        = "bind"
		enableUDP         = "udp"
		udpBufferSize     = "udpBufferSize"
		compatibilityMode = "comp"
	)

	h.md.readTimeout = mdata.GetDuration(md, readTimeout)
	h.md.noTLS = mdata.GetBool(md, noTLS)
	h.md.enableBind = mdata.GetBool(md, enableBind)
	h.md.enableUDP = mdata.GetBool(md, enableUDP)

	if bs := mdata.GetInt(md, udpBufferSize); bs > 0 {
		h.md.udpBufferSize = int(math.Min(math.Max(float64(bs), 512), 64*1024))
	} else {
		h.md.udpBufferSize = 1024
	}

	h.md.compatibilityMode = mdata.GetBool(md, compatibilityMode)

	return nil
}
