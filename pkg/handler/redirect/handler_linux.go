package redirect

import (
	"errors"
	"fmt"
	"net"
	"syscall"
)

func (h *redirectHandler) getOriginalDstAddr(conn net.Conn) (addr net.Addr, c net.Conn, err error) {
	defer conn.Close()

	tc, ok := conn.(*net.TCPConn)
	if !ok {
		err = errors.New("wrong connection type, must be TCP")
		return
	}

	fc, err := tc.File()
	if err != nil {
		return
	}
	defer fc.Close()

	mreq, err := syscall.GetsockoptIPv6Mreq(int(fc.Fd()), syscall.IPPROTO_IP, 80)
	if err != nil {
		return
	}

	// only ipv4 support
	ip := net.IPv4(mreq.Multiaddr[4], mreq.Multiaddr[5], mreq.Multiaddr[6], mreq.Multiaddr[7])
	port := uint16(mreq.Multiaddr[2])<<8 + uint16(mreq.Multiaddr[3])
	addr, err = net.ResolveTCPAddr("tcp4", fmt.Sprintf("%s:%d", ip.String(), port))
	if err != nil {
		return
	}

	c, err = net.FileConn(fc)
	return
}
