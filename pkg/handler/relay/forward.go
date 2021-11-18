package relay

import (
	"context"
	"net"
	"time"

	"github.com/go-gost/gost/pkg/chain"
	"github.com/go-gost/gost/pkg/handler"
)

func (h *relayHandler) handleForward(ctx context.Context, conn net.Conn, network string) {
	target := h.group.Next()
	if target == nil {
		h.logger.Error("no target available")
		return
	}

	h.logger = h.logger.WithFields(map[string]interface{}{
		"dst": target.Addr(),
	})

	h.logger.Infof("%s >> %s", conn.RemoteAddr(), target.Addr())

	r := (&chain.Router{}).
		WithChain(h.chain).
		WithRetry(h.md.retryCount).
		WithLogger(h.logger)

	cc, err := r.Dial(ctx, network, target.Addr())
	if err != nil {
		h.logger.Error(err)
		// TODO: the router itself may be failed due to the failed node in the router,
		// the dead marker may be a wrong operation.
		target.Marker().Mark()
		return
	}
	defer cc.Close()
	target.Marker().Reset()

	t := time.Now()
	h.logger.Infof("%s <-> %s", conn.RemoteAddr(), target.Addr())
	handler.Transport(conn, cc)
	h.logger.
		WithFields(map[string]interface{}{
			"duration": time.Since(t),
		}).
		Infof("%s >-< %s", conn.RemoteAddr(), target.Addr())
}
