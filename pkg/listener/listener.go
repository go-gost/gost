package listener

import (
	"errors"
	"net"

	"github.com/go-gost/gost/v3/pkg/metadata"
)

var (
	ErrClosed = errors.New("accpet on closed listener")
)

// Listener is a server listener, just like a net.Listener.
type Listener interface {
	Init(metadata.Metadata) error
	Accept() (net.Conn, error)
	Addr() net.Addr
	Close() error
}

type AcceptError struct {
	err error
}

func NewAcceptError(err error) error {
	return &AcceptError{err: err}
}

func (e *AcceptError) Error() string {
	return e.err.Error()
}

func (e *AcceptError) Timeout() bool {
	return false
}

func (e *AcceptError) Temporary() bool {
	return true
}

func (e *AcceptError) Unwrap() error {
	return e.err
}
