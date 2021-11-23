package v5

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/go-gost/gosocks5"
	"github.com/go-gost/gost/pkg/common/bufpool"
	"github.com/go-gost/gost/pkg/common/util/socks"
)

func (h *socks5Handler) handleUDPTun(ctx context.Context, conn net.Conn, network, address string) {
	h.logger = h.logger.WithFields(map[string]interface{}{
		"cmd": "udp-tun",
	})

	if !h.md.enableUDP {
		reply := gosocks5.NewReply(gosocks5.NotAllowed, nil)
		reply.Write(conn)
		h.logger.Debug(reply)
		h.logger.Error("UDP relay is diabled")
		return
	}

	bindAddr, _ := net.ResolveUDPAddr(network, address)
	pc, err := net.ListenUDP(network, bindAddr)
	if err != nil {
		h.logger.Error(err)
		return
	}
	defer pc.Close()

	saddr, _ := gosocks5.NewAddr(pc.LocalAddr().String())
	saddr.Host, _, _ = net.SplitHostPort(conn.LocalAddr().String())
	saddr.Type = 0
	reply := gosocks5.NewReply(gosocks5.Succeeded, saddr)
	if err := reply.Write(conn); err != nil {
		h.logger.Error(err)
		return
	}
	h.logger.Debug(reply)

	h.logger = h.logger.WithFields(map[string]interface{}{
		"bind": fmt.Sprintf("%s/%s", pc.LocalAddr(), pc.LocalAddr().Network()),
	})

	h.logger.Debugf("bind on %s OK", pc.LocalAddr())

	t := time.Now()
	h.logger.Infof("%s <-> %s", conn.RemoteAddr(), pc.LocalAddr())
	h.tunnelServerUDP(
		socks.UDPTunServerConn(conn),
		pc,
	)
	h.logger.
		WithFields(map[string]interface{}{
			"duration": time.Since(t),
		}).
		Infof("%s >-< %s", conn.RemoteAddr(), pc.LocalAddr())
}

func (h *socks5Handler) tunnelServerUDP(tunnel, c net.PacketConn) (err error) {
	bufSize := h.md.udpBufferSize
	errc := make(chan error, 2)

	go func() {
		for {
			err := func() error {
				b := bufpool.Get(bufSize)
				defer bufpool.Put(b)

				n, raddr, err := tunnel.ReadFrom(b)
				if err != nil {
					return err
				}

				if h.bypass != nil && h.bypass.Contains(raddr.String()) {
					h.logger.Warn("bypass: ", raddr)
					return nil
				}

				if _, err := c.WriteTo(b[:n], raddr); err != nil {
					return err
				}

				h.logger.Debugf("%s >>> %s data: %d",
					c.LocalAddr(), raddr, n)

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

				n, raddr, err := c.ReadFrom(b)
				if err != nil {
					return err
				}

				if h.bypass != nil && h.bypass.Contains(raddr.String()) {
					h.logger.Warn("bypass: ", raddr)
					return nil
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
