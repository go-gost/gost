package relay

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/go-gost/gost/pkg/handler"
	"github.com/go-gost/relay"
)

func (h *relayHandler) handleForward(ctx context.Context, conn net.Conn, network string) {
	resp := relay.Response{
		Version: relay.Version1,
		Status:  relay.StatusOK,
	}
	target := h.group.Next()
	if target == nil {
		resp.Status = relay.StatusServiceUnavailable
		resp.WriteTo(conn)
		h.logger.Error("no target available")
		return
	}

	h.logger = h.logger.WithFields(map[string]interface{}{
		"dst": fmt.Sprintf("%s/%s", target.Addr(), network),
	})

	h.logger.Infof("%s >> %s", conn.RemoteAddr(), target.Addr())

	cc, err := h.router.Dial(ctx, network, target.Addr())
	if err != nil {
		// TODO: the router itself may be failed due to the failed node in the router,
		// the dead marker may be a wrong operation.
		target.Marker().Mark()

		resp.Status = relay.StatusHostUnreachable
		resp.WriteTo(conn)
		h.logger.Error(err)

		return
	}
	defer cc.Close()
	target.Marker().Reset()

	if h.md.noDelay {
		if _, err := resp.WriteTo(conn); err != nil {
			h.logger.Error(err)
			return
		}
	}

	switch network {
	case "udp", "udp4", "udp6":
		rc := &udpConn{
			Conn: conn,
		}
		if !h.md.noDelay {
			// cache the header
			if _, err := resp.WriteTo(&rc.wbuf); err != nil {
				return
			}
		}
		conn = rc
	default:
		rc := &tcpConn{
			Conn: conn,
		}
		if !h.md.noDelay {
			// cache the header
			if _, err := resp.WriteTo(&rc.wbuf); err != nil {
				return
			}
		}
		conn = rc
	}

	t := time.Now()
	h.logger.Infof("%s <-> %s", conn.RemoteAddr(), target.Addr())
	handler.Transport(conn, cc)
	h.logger.
		WithFields(map[string]interface{}{
			"duration": time.Since(t),
		}).
		Infof("%s >-< %s", conn.RemoteAddr(), target.Addr())
}
