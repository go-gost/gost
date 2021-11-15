package ss

import (
	"context"
	"net"
	"time"

	"github.com/go-gost/gost/pkg/chain"
	"github.com/go-gost/gost/pkg/internal/bufpool"
	"github.com/go-gost/gost/pkg/internal/utils/socks"
	"github.com/go-gost/gost/pkg/internal/utils/ss"
)

func (h *ssHandler) handleUDP(ctx context.Context, raddr net.Addr, conn net.PacketConn) {
	if h.md.cipher != nil {
		conn = h.md.cipher.PacketConn(conn)
	}

	// obtain a udp connection
	r := (&chain.Router{}).
		WithChain(h.chain).
		WithRetry(h.md.retryCount).
		WithLogger(h.logger)
	c, err := r.Dial(ctx, "udp", "")
	if err != nil {
		h.logger.Error(err)
		return
	}

	cc, ok := c.(net.PacketConn)
	if !ok {
		h.logger.Errorf("%s: not a packet connection")
		return
	}
	defer cc.Close()

	h.logger = h.logger.WithFields(map[string]interface{}{
		"bind": cc.LocalAddr().String(),
	})
	h.logger.Infof("bind on %s OK", cc.LocalAddr().String())
	t := time.Now()
	h.logger.Infof("%s <-> %s", raddr, cc.LocalAddr())
	h.relayPacket(
		ss.UDPServerConn(conn, raddr, h.md.bufferSize),
		cc,
	)
	h.logger.
		WithFields(map[string]interface{}{"duration": time.Since(t)}).
		Infof("%s >-< %s", raddr, cc.LocalAddr())
}

func (h *ssHandler) handleUDPTun(ctx context.Context, conn net.Conn) {
	// obtain a udp connection
	r := (&chain.Router{}).
		WithChain(h.chain).
		WithRetry(h.md.retryCount).
		WithLogger(h.logger)
	c, err := r.Dial(ctx, "udp", "")
	if err != nil {
		h.logger.Error(err)
		return
	}

	cc, ok := c.(net.PacketConn)
	if !ok {
		h.logger.Errorf("%s: not a packet connection")
		return
	}
	defer cc.Close()

	h.logger = h.logger.WithFields(map[string]interface{}{
		"bind": cc.LocalAddr().String(),
	})
	h.logger.Infof("bind on %s OK", cc.LocalAddr().String())

	t := time.Now()
	h.logger.Infof("%s <-> %s", conn.RemoteAddr(), cc.LocalAddr())
	h.tunnelUDP(socks.UDPTunServerConn(conn), cc)
	h.logger.
		WithFields(map[string]interface{}{"duration": time.Since(t)}).
		Infof("%s >-< %s", conn.RemoteAddr(), cc.LocalAddr())
}

func (h *ssHandler) relayPacket(pc1, pc2 net.PacketConn) (err error) {
	bufSize := h.md.bufferSize
	errc := make(chan error, 2)

	go func() {
		for {
			err := func() error {
				b := bufpool.Get(bufSize)
				defer bufpool.Put(b)

				n, addr, err := pc1.ReadFrom(b)
				if err != nil {
					return err
				}

				if h.bypass != nil && h.bypass.Contains(addr.String()) {
					h.logger.Warn("bypass: ", addr)
					return nil
				}

				if _, err = pc2.WriteTo(b[:n], addr); err != nil {
					return err
				}

				h.logger.Debugf("%s >>> %s data: %d",
					pc2.LocalAddr(), addr, n)
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

				n, raddr, err := pc2.ReadFrom(b)
				if err != nil {
					return err
				}

				if h.bypass != nil && h.bypass.Contains(raddr.String()) {
					h.logger.Warn("bypass: ", raddr)
					return nil
				}

				if _, err = pc1.WriteTo(b[:n], raddr); err != nil {
					return err
				}

				h.logger.Debugf("%s <<< %s data: %d",
					pc2.LocalAddr(), raddr, n)
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

func (h *ssHandler) tunnelUDP(tunnel, c net.PacketConn) (err error) {
	bufSize := h.md.bufferSize
	errc := make(chan error, 2)

	go func() {
		b := bufpool.Get(bufSize)
		defer bufpool.Put(b)

		for {
			err := func() error {
				n, addr, err := tunnel.ReadFrom(b)
				if err != nil {
					return err
				}

				if h.bypass != nil && h.bypass.Contains(addr.String()) {
					h.logger.Warn("bypass: ", addr.String())
					return nil // bypass
				}

				if _, err := c.WriteTo(b[:n], addr); err != nil {
					return err
				}

				h.logger.Debugf("%s >>> %s data: %d",
					c.LocalAddr(), addr, n)

				return nil
			}()

			if err != nil {
				errc <- err
				return
			}
		}
	}()

	go func() {
		b := bufpool.Get(bufSize)
		defer bufpool.Put(b)

		for {
			err := func() error {
				n, raddr, err := c.ReadFrom(b)
				if err != nil {
					return err
				}

				if h.bypass != nil && h.bypass.Contains(raddr.String()) {
					h.logger.Warn("bypass: ", raddr.String())
					return nil // bypass
				}

				if _, err := tunnel.WriteTo(b[:n], raddr); err != nil {
					return err
				}

				h.logger.Debugf("%s <<< %s data: %d",
					c.LocalAddr(), raddr, n)

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
