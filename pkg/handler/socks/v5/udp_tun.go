package v5

import (
	"context"
	"net"
	"time"

	"github.com/go-gost/gosocks5"
	"github.com/go-gost/gost/pkg/common/util/socks"
	"github.com/go-gost/gost/pkg/handler"
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

	// dummy bind
	reply := gosocks5.NewReply(gosocks5.Succeeded, nil)
	if err := reply.Write(conn); err != nil {
		h.logger.Error(err)
		return
	}
	h.logger.Debug(reply)

	// obtain a udp connection
	c, err := h.router.Dial(ctx, "udp", "") // UDP association
	if err != nil {
		h.logger.Error(err)
		return
	}
	defer c.Close()

	pc, ok := c.(net.PacketConn)
	if !ok {
		h.logger.Errorf("wrong connection type")
		return
	}

	relay := handler.NewUDPRelay(socks.UDPTunServerConn(conn), pc).
		WithBypass(h.bypass).
		WithLogger(h.logger)
	relay.SetBufferSize(h.md.udpBufferSize)

	t := time.Now()
	h.logger.Infof("%s <-> %s", conn.RemoteAddr(), pc.LocalAddr())
	relay.Run()
	h.logger.
		WithFields(map[string]interface{}{
			"duration": time.Since(t),
		}).
		Infof("%s >-< %s", conn.RemoteAddr(), pc.LocalAddr())
}
