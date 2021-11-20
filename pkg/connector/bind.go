package connector

import (
	"context"
	"errors"
	"net"
)

var (
	ErrBindUnsupported = errors.New("bind unsupported")
)

type Accepter interface {
	Accept() (net.Conn, error)
	Addr() net.Addr
	Close() error
}

type Binder interface {
	Bind(ctx context.Context, conn net.Conn, network, address string, opts ...BindOption) (Accepter, error)
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
