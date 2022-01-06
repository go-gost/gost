package handler

import (
	"net"

	"github.com/go-gost/gost/pkg/bypass"
	"github.com/go-gost/gost/pkg/common/bufpool"
	"github.com/go-gost/gost/pkg/logger"
)

type UDPRelay struct {
	pc1 net.PacketConn
	pc2 net.PacketConn

	bypass     bypass.Bypass
	bufferSize int
	logger     logger.Logger
}

func NewUDPRelay(pc1, pc2 net.PacketConn) *UDPRelay {
	return &UDPRelay{
		pc1: pc1,
		pc2: pc2,
	}
}

func (r *UDPRelay) WithBypass(bp bypass.Bypass) *UDPRelay {
	r.bypass = bp
	return r
}

func (r *UDPRelay) WithLogger(logger logger.Logger) *UDPRelay {
	r.logger = logger
	return r
}

func (r *UDPRelay) SetBufferSize(n int) {
	r.bufferSize = n
}

func (r *UDPRelay) Run() (err error) {
	bufSize := r.bufferSize
	if bufSize <= 0 {
		bufSize = 1024
	}

	errc := make(chan error, 2)

	go func() {
		for {
			err := func() error {
				b := bufpool.Get(bufSize)
				defer bufpool.Put(b)

				n, raddr, err := r.pc1.ReadFrom(*b)
				if err != nil {
					return err
				}

				if r.bypass != nil && r.bypass.Contains(raddr.String()) {
					if r.logger != nil {
						r.logger.Warn("bypass: ", raddr)
					}
					return nil
				}

				if _, err := r.pc2.WriteTo((*b)[:n], raddr); err != nil {
					return err
				}

				if r.logger != nil {
					r.logger.Debugf("%s >>> %s data: %d",
						r.pc2.LocalAddr(), raddr, n)

				}

				return nil
			}()

			if err != nil {
				errc <- err
				return
			}
		}
	}()

	go func() {
		for {
			err := func() error {
				b := bufpool.Get(bufSize)
				defer bufpool.Put(b)

				n, raddr, err := r.pc2.ReadFrom(*b)
				if err != nil {
					return err
				}

				if r.bypass != nil && r.bypass.Contains(raddr.String()) {
					if r.logger != nil {
						r.logger.Warn("bypass: ", raddr)
					}
					return nil
				}

				if _, err := r.pc1.WriteTo((*b)[:n], raddr); err != nil {
					return err
				}

				if r.logger != nil {
					r.logger.Debugf("%s <<< %s data: %d",
						r.pc2.LocalAddr(), raddr, n)

				}

				return nil
			}()

			if err != nil {
				errc <- err
				return
			}
		}
	}()

	return <-errc
}
