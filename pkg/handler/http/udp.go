package http

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/go-gost/gost/v3/pkg/common/net/relay"
	"github.com/go-gost/gost/v3/pkg/internal/util/socks"
	"github.com/go-gost/gost/v3/pkg/logger"
)

func (h *httpHandler) handleUDP(ctx context.Context, conn net.Conn, network, address string, log logger.Logger) error {
	log = log.WithFields(map[string]any{
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

		if log.IsLevelEnabled(logger.DebugLevel) {
			dump, _ := httputil.DumpResponse(resp, false)
			log.Debug(string(dump))
		}

		log.Error("http: UDP relay is disabled")

		return resp.Write(conn)
	}

	resp.StatusCode = http.StatusOK
	if log.IsLevelEnabled(logger.DebugLevel) {
		dump, _ := httputil.DumpResponse(resp, false)
		log.Debug(string(dump))
	}
	if err := resp.Write(conn); err != nil {
		log.Error(err)
		return err
	}

	// obtain a udp connection
	c, err := h.router.Dial(ctx, "udp", "") // UDP association
	if err != nil {
		log.Error(err)
		return err
	}
	defer c.Close()

	pc, ok := c.(net.PacketConn)
	if !ok {
		err = errors.New("wrong connection type")
		log.Error(err)
		return err
	}

	relay := relay.NewUDPRelay(socks.UDPTunServerConn(conn), pc).
		WithBypass(h.options.Bypass).
		WithLogger(log)

	t := time.Now()
	log.Infof("%s <-> %s", conn.RemoteAddr(), pc.LocalAddr())
	relay.Run()
	log.WithFields(map[string]any{
		"duration": time.Since(t),
	}).Infof("%s >-< %s", conn.RemoteAddr(), pc.LocalAddr())

	return nil
}
