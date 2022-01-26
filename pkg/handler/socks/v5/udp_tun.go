package v5

import (
	"context"
	"net"
	"time"

	"github.com/go-gost/gosocks5"
	"github.com/go-gost/gost/pkg/common/util/socks"
	"github.com/go-gost/gost/pkg/handler"
	"github.com/go-gost/gost/pkg/logger"
)

func (h *socks5Handler) handleUDPTun(ctx context.Context, conn net.Conn, network, address string, log logger.Logger) {
	log = log.WithFields(map[string]interface{}{
		"cmd": "udp-tun",
	})

	if !h.md.enableUDP {
		reply := gosocks5.NewReply(gosocks5.NotAllowed, nil)
		reply.Write(conn)
		log.Debug(reply)
		log.Error("UDP relay is diabled")
		return
	}

	// dummy bind
	reply := gosocks5.NewReply(gosocks5.Succeeded, nil)
	if err := reply.Write(conn); err != nil {
		log.Error(err)
		return
	}
	log.Debug(reply)

	// obtain a udp connection
	c, err := h.router.Dial(ctx, "udp", "") // UDP association
	if err != nil {
		log.Error(err)
		return
	}
	defer c.Close()

	pc, ok := c.(net.PacketConn)
	if !ok {
		log.Errorf("wrong connection type")
		return
	}

	relay := handler.NewUDPRelay(socks.UDPTunServerConn(conn), pc).
		WithBypass(h.options.Bypass).
		WithLogger(log)
	relay.SetBufferSize(h.md.udpBufferSize)

	t := time.Now()
	log.Infof("%s <-> %s", conn.RemoteAddr(), pc.LocalAddr())
	relay.Run()
	log.WithFields(map[string]interface{}{
		"duration": time.Since(t),
	}).Infof("%s >-< %s", conn.RemoteAddr(), pc.LocalAddr())
}
