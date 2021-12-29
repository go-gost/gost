package resolver

import (
	"time"
)

type NameServer struct {
	Addr      string
	Protocol  string
	Hostname  string // for TLS handshake verification
	Exchanger Exchanger
	Timeout   time.Duration
}
