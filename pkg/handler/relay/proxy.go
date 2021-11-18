package relay

import (
	"context"
	"net"
	"time"

	"github.com/go-gost/gost/pkg/chain"
	util_relay "github.com/go-gost/gost/pkg/common/util/relay"
	"github.com/go-gost/gost/pkg/handler"
	"github.com/go-gost/relay"
)

func (h *relayHandler) handleProxy(ctx context.Context, conn net.Conn, network, address string) {
	h.logger.Infof("%s >> %s", conn.RemoteAddr(), address)

	resp := relay.Response{
		Version: relay.Version1,
		Status:  relay.StatusOK,
	}

	if h.bypass != nil && h.bypass.Contains(address) {
		h.logger.Info("bypass: ", address)
		resp.Status = relay.StatusForbidden
		resp.WriteTo(conn)
		return
	}

	r := (&chain.Router{}).
		WithChain(h.chain).
		WithRetry(h.md.retryCount).
		WithLogger(h.logger)
	cc, err := r.Dial(ctx, network, address)
	if err != nil {
		resp.Status = relay.StatusNetworkUnreachable
		resp.WriteTo(conn)
		return
	}
	defer cc.Close()

	if _, err := resp.WriteTo(conn); err != nil {
		h.logger.Error(err)
	}

	if network == "udp" {
		conn = util_relay.UDPTunConn(conn)
	}

	t := time.Now()
	h.logger.Infof("%s <-> %s", conn.RemoteAddr(), address)
	handler.Transport(conn, cc)
	h.logger.
		WithFields(map[string]interface{}{
			"duration": time.Since(t),
		}).
		Infof("%s >-< %s", conn.RemoteAddr(), address)
}
