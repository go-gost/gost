package v5

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/go-gost/gosocks5"
	"github.com/go-gost/gost/pkg/handler"
	"github.com/go-gost/gost/pkg/logger"
)

func (h *socks5Handler) handleConnect(ctx context.Context, conn net.Conn, network, address string, log logger.Logger) {
	log = log.WithFields(map[string]interface{}{
		"dst": fmt.Sprintf("%s/%s", address, network),
		"cmd": "connect",
	})
	log.Infof("%s >> %s", conn.RemoteAddr(), address)

	if h.options.Bypass != nil && h.options.Bypass.Contains(address) {
		resp := gosocks5.NewReply(gosocks5.NotAllowed, nil)
		resp.Write(conn)
		log.Debug(resp)
		log.Info("bypass: ", address)
		return
	}

	cc, err := h.router.Dial(ctx, network, address)
	if err != nil {
		resp := gosocks5.NewReply(gosocks5.NetUnreachable, nil)
		resp.Write(conn)
		log.Debug(resp)
		return
	}

	defer cc.Close()

	resp := gosocks5.NewReply(gosocks5.Succeeded, nil)
	if err := resp.Write(conn); err != nil {
		log.Error(err)
		return
	}
	log.Debug(resp)

	t := time.Now()
	log.Infof("%s <-> %s", conn.RemoteAddr(), address)
	handler.Transport(conn, cc)
	log.WithFields(map[string]interface{}{
		"duration": time.Since(t),
	}).Infof("%s >-< %s", conn.RemoteAddr(), address)
}
