package v5

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/go-gost/gosocks5"
	"github.com/go-gost/gost/pkg/chain"
	"github.com/go-gost/gost/pkg/handler"
)

func (h *socks5Handler) handleConnect(ctx context.Context, conn net.Conn, network, address string) {
	h.logger = h.logger.WithFields(map[string]interface{}{
		"dst": fmt.Sprintf("%s/%s", address, network),
		"cmd": "connect",
	})
	h.logger.Infof("%s >> %s", conn.RemoteAddr(), address)

	if h.bypass != nil && h.bypass.Contains(address) {
		resp := gosocks5.NewReply(gosocks5.NotAllowed, nil)
		resp.Write(conn)
		h.logger.Debug(resp)
		h.logger.Info("bypass: ", address)
		return
	}

	r := (&chain.Router{}).
		WithChain(h.chain).
		WithRetry(h.md.retryCount).
		WithLogger(h.logger)
	cc, err := r.Dial(ctx, network, address)
	if err != nil {
		resp := gosocks5.NewReply(gosocks5.NetUnreachable, nil)
		resp.Write(conn)
		h.logger.Debug(resp)
		return
	}

	defer cc.Close()

	resp := gosocks5.NewReply(gosocks5.Succeeded, nil)
	if err := resp.Write(conn); err != nil {
		h.logger.Error(err)
		return
	}
	h.logger.Debug(resp)

	t := time.Now()
	h.logger.Infof("%s <-> %s", conn.RemoteAddr(), address)
	handler.Transport(conn, cc)
	h.logger.
		WithFields(map[string]interface{}{
			"duration": time.Since(t),
		}).
		Infof("%s >-< %s", conn.RemoteAddr(), address)
}
