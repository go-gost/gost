//go:build !linux

package udp

import (
	"errors"
	"net"
)

func (l *redirectListener) listenUDP(addr *net.UDPAddr) (*net.UDPConn, error) {
	return nil, errors.New("UDP redirect is not available on non-linux platform")
}

func (l *redirectListener) accept() (conn net.Conn, err error) {
	return nil, errors.New("UDP redirect is not available on non-linux platform")
}
