package v5

import (
	"crypto/tls"
	"net/url"
	"strings"
	"time"

	md "github.com/go-gost/gost/pkg/metadata"
)

const (
	defaultTTL            = 60 * time.Second
	defaultReadBufferSize = 4096
	defaultReadQueueSize  = 128
	defaultBacklog        = 128
)

type metadata struct {
	connectTimeout time.Duration
	User           *url.Userinfo
	tlsConfig      *tls.Config
	noTLS          bool

	ttl            time.Duration
	readBufferSize int
	readQueueSize  int
	backlog        int
}

func (c *socks5Connector) parseMetadata(md md.Metadata) (err error) {
	const (
		connectTimeout = "timeout"
		auth           = "auth"
		noTLS          = "notls"

		ttl            = "ttl"
		readBufferSize = "readBufferSize"
		readQueueSize  = "readQueueSize"
		backlog        = "backlog"
	)

	if v := md.GetString(auth); v != "" {
		ss := strings.SplitN(v, ":", 2)
		if len(ss) == 1 {
			c.md.User = url.User(ss[0])
		} else {
			c.md.User = url.UserPassword(ss[0], ss[1])
		}
	}

	c.md.connectTimeout = md.GetDuration(connectTimeout)
	c.md.noTLS = md.GetBool(noTLS)

	c.md.ttl = md.GetDuration(ttl)
	if c.md.ttl <= 0 {
		c.md.ttl = defaultTTL
	}
	c.md.readBufferSize = md.GetInt(readBufferSize)
	if c.md.readBufferSize <= 0 {
		c.md.readBufferSize = defaultReadBufferSize
	}

	c.md.readQueueSize = md.GetInt(readQueueSize)
	if c.md.readQueueSize <= 0 {
		c.md.readQueueSize = defaultReadQueueSize
	}

	c.md.backlog = md.GetInt(backlog)
	if c.md.backlog <= 0 {
		c.md.backlog = defaultBacklog
	}
	return
}
