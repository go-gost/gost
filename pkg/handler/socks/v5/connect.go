package v5

import (
	"context"
	"net"
	"time"

	"github.com/go-gost/gosocks5"
	"github.com/go-gost/gost/pkg/handler"
	"github.com/go-gost/gost/pkg/logger"
)

func (h *socks5Handler) handleConnect(ctx context.Context, conn net.Conn, addr string) {
	h.logger = h.logger.WithFields(map[string]interface{}{
		"dst": addr,
		"cmd": "connect",
	})
	h.logger.Infof("%s >> %s", conn.RemoteAddr(), addr)

	if h.bypass != nil && h.bypass.Contains(addr) {
		resp := gosocks5.NewReply(gosocks5.NotAllowed, nil)
		resp.Write(conn)
		if h.logger.IsLevelEnabled(logger.DebugLevel) {
			h.logger.Debug(resp)
		}
		h.logger.Info("bypass: ", addr)
		return
	}

	r := (&handler.Router{}).
		WithChain(h.chain).
		WithRetry(h.md.retryCount).
		WithLogger(h.logger)
	cc, err := r.Dial(ctx, "tcp", addr)
	if err != nil {
		resp := gosocks5.NewReply(gosocks5.NetUnreachable, nil)
		resp.Write(conn)
		if h.logger.IsLevelEnabled(logger.DebugLevel) {
			h.logger.Debug(resp)
		}
		return
	}

	defer cc.Close()

	resp := gosocks5.NewReply(gosocks5.Succeeded, nil)
	if err := resp.Write(conn); err != nil {
		h.logger.Error(err)
		return
	}
	if h.logger.IsLevelEnabled(logger.DebugLevel) {
		h.logger.Debug(resp)
	}

	t := time.Now()
	h.logger.Infof("%s <-> %s", conn.RemoteAddr(), addr)
	handler.Transport(conn, cc)
	h.logger.
		WithFields(map[string]interface{}{
			"duration": time.Since(t),
		}).
		Infof("%s >-< %s", conn.RemoteAddr(), addr)
}
