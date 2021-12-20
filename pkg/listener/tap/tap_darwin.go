package tap

import (
	"errors"
	"net"

	"github.com/songgao/water"
)

func (l *tapListener) createTap() (ifce *water.Interface, ip net.IP, err error) {
	err = errors.New("tap is not supported on darwin")
	return
}
