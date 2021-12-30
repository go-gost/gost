package http

import (
	"context"
	"net"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/go-gost/gost/pkg/common/util/socks"
	"github.com/go-gost/gost/pkg/handler"
	"github.com/go-gost/gost/pkg/logger"
)

func (h *httpHandler) handleUDP(ctx context.Context, conn net.Conn, network, address string) {
	h.logger = h.logger.WithFields(map[string]interface{}{
		"cmd": "udp",
	})

	resp := &http.Response{
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     h.md.header,
	}
	if resp.Header == nil {
		resp.Header = http.Header{}
	}

	if !h.md.enableUDP {
		resp.StatusCode = http.StatusForbidden
		resp.Write(conn)

		if h.logger.IsLevelEnabled(logger.DebugLevel) {
			dump, _ := httputil.DumpResponse(resp, false)
			h.logger.Debug(string(dump))
		}
		h.logger.Error("UDP relay is diabled")

		return
	}

	resp.StatusCode = http.StatusOK
	if h.logger.IsLevelEnabled(logger.DebugLevel) {
		dump, _ := httputil.DumpResponse(resp, false)
		h.logger.Debug(string(dump))
	}
	if err := resp.Write(conn); err != nil {
		h.logger.Error(err)
		return
	}

	// obtain a udp connection
	c, err := h.router.Dial(ctx, "udp", "") // UDP association
	if err != nil {
		h.logger.Error(err)
		return
	}
	defer c.Close()

	pc, ok := c.(net.PacketConn)
	if !ok {
		h.logger.Errorf("wrong connection type")
		return
	}

	relay := handler.NewUDPRelay(socks.UDPTunServerConn(conn), pc).
		WithBypass(h.bypass).
		WithLogger(h.logger)

	t := time.Now()
	h.logger.Infof("%s <-> %s", conn.RemoteAddr(), pc.LocalAddr())
	relay.Run()
	h.logger.
		WithFields(map[string]interface{}{
			"duration": time.Since(t),
		}).
		Infof("%s >-< %s", conn.RemoteAddr(), pc.LocalAddr())
}
