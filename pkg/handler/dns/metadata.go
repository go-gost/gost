package dns

import (
	"time"

	mdata "github.com/go-gost/gost/pkg/metadata"
)

type metadata struct {
	readTimeout time.Duration
	retryCount  int
	ttl         time.Duration
	timeout     time.Duration
	prefer      string
	clientIP    string
	// nameservers
	servers []string
}

func (h *dnsHandler) parseMetadata(md mdata.Metadata) (err error) {
	const (
		readTimeout = "readTimeout"
		retryCount  = "retry"
		ttl         = "ttl"
		timeout     = "timeout"
		prefer      = "prefer"
		clientIP    = "clientIP"
		servers     = "servers"
	)

	h.md.readTimeout = mdata.GetDuration(md, readTimeout)
	h.md.retryCount = mdata.GetInt(md, retryCount)
	h.md.ttl = mdata.GetDuration(md, ttl)
	h.md.timeout = mdata.GetDuration(md, timeout)
	if h.md.timeout <= 0 {
		h.md.timeout = 5 * time.Second
	}
	h.md.prefer = mdata.GetString(md, prefer)
	h.md.clientIP = mdata.GetString(md, clientIP)
	h.md.servers = mdata.GetStrings(md, servers)

	return
}
