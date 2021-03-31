package utils

import (
	"net"
	"time"
)

const (
	defaultKeepAlivePeriod = 180 * time.Second
)

// TCPKeepAliveListener is a TCP listener with keep alive enabled.
type TCPKeepAliveListener struct {
	KeepAlivePeriod time.Duration
	*net.TCPListener
}

func (l *TCPKeepAliveListener) Accept() (c net.Conn, err error) {
	tc, err := l.AcceptTCP()
	if err != nil {
		return
	}

	tc.SetKeepAlive(true)
	period := l.KeepAlivePeriod
	if period <= 0 {
		period = defaultKeepAlivePeriod
	}
	tc.SetKeepAlivePeriod(period)

	return tc, nil
}
