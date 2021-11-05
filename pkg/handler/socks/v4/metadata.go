package v4

import (
	"time"

	"github.com/go-gost/gost/pkg/auth"
)

const (
	authsKey    = "auths"
	readTimeout = "readTimeout"
	retryCount  = "retry"
)

type metadata struct {
	authenticator auth.Authenticator
	readTimeout   time.Duration
	retryCount    int
}
