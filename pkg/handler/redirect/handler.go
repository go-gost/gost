package redirect

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/go-gost/gost/pkg/bypass"
	"github.com/go-gost/gost/pkg/chain"
	"github.com/go-gost/gost/pkg/handler"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
)

func init() {
	registry.RegisterHandler("red", NewHandler)
	registry.RegisterHandler("redu", NewHandler)
	registry.RegisterHandler("redir", NewHandler)
	registry.RegisterHandler("redirect", NewHandler)
}

type redirectHandler struct {
	bypass bypass.Bypass
	router *chain.Router
	logger logger.Logger
	md     metadata
}

func NewHandler(opts ...handler.Option) handler.Handler {
	options := &handler.Options{}
	for _, opt := range opts {
		opt(options)
	}

	return &redirectHandler{
		bypass: options.Bypass,
		router: options.Router,
		logger: options.Logger,
	}
}

func (h *redirectHandler) Init(md md.Metadata) (err error) {
	return h.parseMetadata(md)
}

func (h *redirectHandler) Handle(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	start := time.Now()
	h.logger = h.logger.WithFields(map[string]interface{}{
		"remote": conn.RemoteAddr().String(),
		"local":  conn.LocalAddr().String(),
	})

	h.logger.Infof("%s <> %s", conn.RemoteAddr(), conn.LocalAddr())
	defer func() {
		h.logger.WithFields(map[string]interface{}{
			"duration": time.Since(start),
		}).Infof("%s >< %s", conn.RemoteAddr(), conn.LocalAddr())
	}()

	network := "tcp"
	var dstAddr net.Addr
	var err error

	if _, ok := conn.(net.PacketConn); ok {
		network = "udp"
		dstAddr = conn.LocalAddr()
	}

	if network == "tcp" {
		dstAddr, conn, err = h.getOriginalDstAddr(conn)
		if err != nil {
			h.logger.Error(err)
			return
		}
	}

	h.logger = h.logger.WithFields(map[string]interface{}{
		"dst": fmt.Sprintf("%s/%s", dstAddr, network),
	})

	h.logger.Infof("%s >> %s", conn.RemoteAddr(), dstAddr)

	if h.bypass != nil && h.bypass.Contains(dstAddr.String()) {
		h.logger.Info("bypass: ", dstAddr)
		return
	}

	cc, err := h.router.Dial(ctx, network, dstAddr.String())
	if err != nil {
		h.logger.Error(err)
		return
	}
	defer cc.Close()

	t := time.Now()
	h.logger.Infof("%s <-> %s", conn.RemoteAddr(), dstAddr)
	handler.Transport(conn, cc)
	h.logger.
		WithFields(map[string]interface{}{
			"duration": time.Since(t),
		}).
		Infof("%s >-< %s", conn.RemoteAddr(), dstAddr)
}
