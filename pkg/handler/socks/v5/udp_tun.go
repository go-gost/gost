package v5

import (
	"context"
	"net"
	"time"

	"github.com/go-gost/gosocks5"
	"github.com/go-gost/gost/pkg/handler"
	"github.com/go-gost/gost/pkg/internal/bufpool"
	"github.com/go-gost/gost/pkg/logger"
)

func (h *socks5Handler) handleUDPTun(ctx context.Context, conn net.Conn, req *gosocks5.Request) {
	h.logger = h.logger.WithFields(map[string]interface{}{
		"cmd": "udp-tun",
	})

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
		if h.logger.IsLevelEnabled(logger.DebugLevel) {
			h.logger.Debug(reply)
		}

		h.logger = h.logger.WithFields(map[string]interface{}{
			"bind": saddr.String(),
		})

		t := time.Now()
		h.logger.Infof("%s <-> %s", conn.RemoteAddr(), saddr)
		h.tunnelServerUDP(conn, relay)
		h.logger.
			WithFields(map[string]interface{}{
				"duration": time.Since(t),
			}).
			Infof("%s >-< %s", conn.RemoteAddr(), saddr)

		return
	}

	r := (&handler.Router{}).
		WithChain(h.chain).
		WithRetry(h.md.retryCount).
		WithLogger(h.logger)
	cc, err := r.Connect(ctx)
	if err != nil {
		h.logger.Error(err)
		reply := gosocks5.NewReply(gosocks5.Failure, nil)
		reply.Write(conn)
		if h.logger.IsLevelEnabled(logger.DebugLevel) {
			h.logger.Debug(reply)
		}
		return
	}
	defer cc.Close()

	// forward request
	if err := req.Write(cc); err != nil {
		h.logger.Error(err)
		reply := gosocks5.NewReply(gosocks5.Failure, nil)
		reply.Write(conn)
		if h.logger.IsLevelEnabled(logger.DebugLevel) {
			h.logger.Debug(reply)
		}
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

func (h *socks5Handler) tunnelServerUDP(tunnel net.Conn, c net.PacketConn) (err error) {
	bufSize := h.md.udpBufferSize
	errc := make(chan error, 2)

	go func() {
		b := bufpool.Get(bufSize)
		defer bufpool.Put(b)

		const dataPos = 262

		for {
			addr := gosocks5.Addr{}
			header := gosocks5.UDPHeader{
				Addr: &addr,
			}

			data := b[dataPos:]
			dgram := gosocks5.UDPDatagram{
				Header: &header,
				Data:   data,
			}
			_, err := dgram.ReadFrom(tunnel)
			if err != nil {
				errc <- err
				return
			}
			// NOTE: the dgram.Data may be reallocated if the provided buffer is too short,
			// we drop it for simplicity. As this occurs, you should enlarge the buffer size.
			if len(dgram.Data) > len(data) {
				h.logger.Warnf("buffer too short, dropped")
				continue
			}

			raddr, err := net.ResolveUDPAddr("udp", addr.String())
			if err != nil {
				continue // drop silently
			}
			if h.bypass != nil && h.bypass.Contains(raddr.String()) {
				h.logger.Warn("bypass: ", raddr.String())
				continue // bypass
			}

			if _, err := c.WriteTo(dgram.Data, raddr); err != nil {
				errc <- err
				return
			}

			if h.logger.IsLevelEnabled(logger.DebugLevel) {
				h.logger.Debugf("%s >>> %s: %v data: %d",
					tunnel.RemoteAddr(), raddr, header.String(), len(dgram.Data))
			}
		}
	}()

	go func() {
		b := bufpool.Get(bufSize)
		defer bufpool.Put(b)

		for {
			n, raddr, err := c.ReadFrom(b)
			if err != nil {
				errc <- err
				return
			}

			if h.bypass != nil && h.bypass.Contains(raddr.String()) {
				h.logger.Warn("bypass: ", raddr.String())
				continue // bypass
			}

			addr, _ := gosocks5.NewAddr(raddr.String())
			if addr == nil {
				addr = &gosocks5.Addr{}
			}
			header := gosocks5.UDPHeader{
				Rsv:  uint16(n),
				Addr: addr,
			}
			dgram := gosocks5.UDPDatagram{
				Header: &header,
				Data:   b[:n],
			}

			if _, err := dgram.WriteTo(tunnel); err != nil {
				errc <- err
				return
			}
			if h.logger.IsLevelEnabled(logger.DebugLevel) {
				h.logger.Debugf("%s <<< %s: %v data: %d",
					tunnel.RemoteAddr(), raddr, header.String(), len(dgram.Data))
			}
		}
	}()

	return <-errc
}
