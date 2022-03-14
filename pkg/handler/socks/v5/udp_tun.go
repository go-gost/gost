package v5

import (
	"context"
	"net"
	"time"

	"github.com/go-gost/gosocks5"
	"github.com/go-gost/gost/v3/pkg/common/net/relay"
	"github.com/go-gost/gost/v3/pkg/internal/util/socks"
	"github.com/go-gost/gost/v3/pkg/logger"
)

func (h *socks5Handler) handleUDPTun(ctx context.Context, conn net.Conn, network, address string, log logger.Logger) error {
	log = log.WithFields(map[string]any{
		"cmd": "udp-tun",
	})

	bindAddr, _ := net.ResolveUDPAddr(network, address)
	if bindAddr == nil {
		bindAddr = &net.UDPAddr{}
	}

	if bindAddr.Port == 0 {
		// relay mode
		if !h.md.enableUDP {
			reply := gosocks5.NewReply(gosocks5.NotAllowed, nil)
			log.Debug(reply)
			log.Error("socks5: UDP relay is disabled")
			return reply.Write(conn)
		}
	} else {
		// BIND mode
		if !h.md.enableBind {
			reply := gosocks5.NewReply(gosocks5.NotAllowed, nil)
			log.Debug(reply)
			log.Error("socks5: BIND is disabled")
			return reply.Write(conn)
		}
	}

	pc, err := net.ListenUDP(network, bindAddr)
	if err != nil {
		log.Error(err)
		return err
	}
	defer pc.Close()

	saddr := gosocks5.Addr{}
	saddr.ParseFrom(pc.LocalAddr().String())
	reply := gosocks5.NewReply(gosocks5.Succeeded, &saddr)
	if err := reply.Write(conn); err != nil {
		log.Error(err)
		return err
	}
	log.Debug(reply)
	log.Debugf("bind on %s OK", pc.LocalAddr())

	r := relay.NewUDPRelay(socks.UDPTunServerConn(conn), pc).
		WithBypass(h.options.Bypass).
		WithLogger(log)
	r.SetBufferSize(h.md.udpBufferSize)

	t := time.Now()
	log.Infof("%s <-> %s", conn.RemoteAddr(), pc.LocalAddr())
	r.Run()
	log.WithFields(map[string]any{
		"duration": time.Since(t),
	}).Infof("%s >-< %s", conn.RemoteAddr(), pc.LocalAddr())

	return nil
}
