package v5

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/go-gost/gosocks5"
	netpkg "github.com/go-gost/gost/pkg/common/net"
	"github.com/go-gost/gost/pkg/logger"
)

func (h *socks5Handler) handleConnect(ctx context.Context, conn net.Conn, network, address string, log logger.Logger) error {
	log = log.WithFields(map[string]any{
		"dst": fmt.Sprintf("%s/%s", address, network),
		"cmd": "connect",
	})
	log.Infof("%s >> %s", conn.RemoteAddr(), address)

	if h.options.Bypass != nil && h.options.Bypass.Contains(address) {
		resp := gosocks5.NewReply(gosocks5.NotAllowed, nil)
		log.Debug(resp)
		log.Info("bypass: ", address)
		return resp.Write(conn)
	}

	cc, err := h.router.Dial(ctx, network, address)
	if err != nil {
		resp := gosocks5.NewReply(gosocks5.NetUnreachable, nil)
		log.Debug(resp)
		resp.Write(conn)
		return err
	}

	defer cc.Close()

	resp := gosocks5.NewReply(gosocks5.Succeeded, nil)
	if err := resp.Write(conn); err != nil {
		log.Error(err)
		return err
	}
	log.Debug(resp)

	t := time.Now()
	log.Infof("%s <-> %s", conn.RemoteAddr(), address)
	netpkg.Transport(conn, cc)
	log.WithFields(map[string]any{
		"duration": time.Since(t),
	}).Infof("%s >-< %s", conn.RemoteAddr(), address)

	return nil
}
