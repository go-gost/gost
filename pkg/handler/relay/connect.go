package relay

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/go-gost/gost/pkg/handler"
	"github.com/go-gost/relay"
)

func (h *relayHandler) handleConnect(ctx context.Context, conn net.Conn, network, address string) {
	h.logger = h.logger.WithFields(map[string]interface{}{
		"dst": fmt.Sprintf("%s/%s", address, network),
		"cmd": "connect",
	})

	h.logger.Infof("%s >> %s", conn.RemoteAddr(), address)

	resp := relay.Response{
		Version: relay.Version1,
		Status:  relay.StatusOK,
	}

	if address == "" {
		resp.Status = relay.StatusBadRequest
		resp.WriteTo(conn)
		h.logger.Error("target not specified")
		return
	}

	if h.bypass != nil && h.bypass.Contains(address) {
		h.logger.Info("bypass: ", address)
		resp.Status = relay.StatusForbidden
		resp.WriteTo(conn)
		return
	}

	cc, err := h.router.Dial(ctx, network, address)
	if err != nil {
		resp.Status = relay.StatusNetworkUnreachable
		resp.WriteTo(conn)
		return
	}
	defer cc.Close()

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
	h.logger.Infof("%s <-> %s", conn.RemoteAddr(), address)
	handler.Transport(conn, cc)
	h.logger.
		WithFields(map[string]interface{}{
			"duration": time.Since(t),
		}).
		Infof("%s >-< %s", conn.RemoteAddr(), address)
}
