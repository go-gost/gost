//go:build !linux

package redirect

import (
	"errors"
	"net"
)

func (h *redirectHandler) getOriginalDstAddr(conn net.Conn) (addr net.Addr, c net.Conn, err error) {
	defer conn.Close()

	err = errors.New("TCP redirect is not available on non-linux platform")
	return
}
