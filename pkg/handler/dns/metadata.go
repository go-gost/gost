package dns

import (
	"net"
	"strings"
	"time"

	mdata "github.com/go-gost/gost/pkg/metadata"
)

type metadata struct {
	readTimeout time.Duration
	retryCount  int
	ttl         time.Duration
	timeout     time.Duration
	clientIP    net.IP
	// nameservers
	servers []string
	dns     []string // compatible with v2
}

func (h *dnsHandler) parseMetadata(md mdata.Metadata) (err error) {
	const (
		readTimeout = "readTimeout"
		retryCount  = "retry"
		ttl         = "ttl"
		timeout     = "timeout"
		clientIP    = "clientIP"
		servers     = "servers"
		dns         = "dns"
	)

	h.md.readTimeout = mdata.GetDuration(md, readTimeout)
	h.md.retryCount = mdata.GetInt(md, retryCount)
	h.md.ttl = mdata.GetDuration(md, ttl)
	h.md.timeout = mdata.GetDuration(md, timeout)
	if h.md.timeout <= 0 {
		h.md.timeout = 5 * time.Second
	}
	sip := mdata.GetString(md, clientIP)
	if sip != "" {
		h.md.clientIP = net.ParseIP(sip)
	}
	h.md.servers = mdata.GetStrings(md, servers)
	h.md.dns = strings.Split(mdata.GetString(md, dns), ",")
	if len(h.md.dns) > 0 {
		h.md.servers = append(h.md.servers, h.md.dns...)
	}

	return
}
