package v5

import (
	"context"
	"net"
	"time"

	"github.com/go-gost/gosocks5"
	"github.com/go-gost/gost/pkg/chain"
	"github.com/go-gost/gost/pkg/common/bufpool"
	"github.com/go-gost/gost/pkg/common/util/socks"
	"github.com/go-gost/gost/pkg/handler"
)

func (h *socks5Handler) handleUDPTun(ctx context.Context, conn net.Conn, req *gosocks5.Request) {
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

	if h.chain.IsEmpty() {
		addr := req.Addr.String()

		bindAddr, _ := net.ResolveUDPAddr("udp", addr)
		relay, err := net.ListenUDP("udp", bindAddr)
		if err != nil {
			h.logger.Error(err)
			return
		}
		defer relay.Close()

		saddr, _ := gosocks5.NewAddr(relay.LocalAddr().String())
		saddr.Host, _, _ = net.SplitHostPort(conn.LocalAddr().String())
		saddr.Type = 0
		reply := gosocks5.NewReply(gosocks5.Succeeded, saddr)
		if err := reply.Write(conn); err != nil {
			h.logger.Error(err)
			return
		}
		h.logger.Debug(reply)

		h.logger = h.logger.WithFields(map[string]interface{}{
			"bind": saddr.String(),
		})

		t := time.Now()
		h.logger.Infof("%s <-> %s", conn.RemoteAddr(), saddr)
		h.tunnelServerUDP(
			socks.UDPTunServerConn(conn),
			relay,
		)
		h.logger.
			WithFields(map[string]interface{}{
				"duration": time.Since(t),
			}).
			Infof("%s >-< %s", conn.RemoteAddr(), saddr)

		return
	}

	r := (&chain.Router{}).
		WithChain(h.chain).
		WithRetry(h.md.retryCount).
		WithLogger(h.logger)
	cc, err := r.Connect(ctx)
	if err != nil {
		h.logger.Error(err)
		reply := gosocks5.NewReply(gosocks5.Failure, nil)
		reply.Write(conn)
		h.logger.Debug(reply)
		return
	}
	defer cc.Close()

	// forward request
	if err := req.Write(cc); err != nil {
		h.logger.Error(err)
		reply := gosocks5.NewReply(gosocks5.Failure, nil)
		reply.Write(conn)
		h.logger.Debug(reply)
	}

	t := time.Now()
	h.logger.Infof("%s <-> %s", conn.RemoteAddr(), cc.RemoteAddr())
	handler.Transport(conn, cc)
	h.logger.
		WithFields(map[string]interface{}{
			"duration": time.Since(t),
		}).
		Infof("%s >-< %s", conn.RemoteAddr(), cc.RemoteAddr())
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
