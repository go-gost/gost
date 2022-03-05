package redirect

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/go-gost/gost/pkg/chain"
	netpkg "github.com/go-gost/gost/pkg/common/net"
	"github.com/go-gost/gost/pkg/handler"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
)

func init() {
	registry.HandlerRegistry().Register("red", NewHandler)
	registry.HandlerRegistry().Register("redu", NewHandler)
	registry.HandlerRegistry().Register("redir", NewHandler)
	registry.HandlerRegistry().Register("redirect", NewHandler)
}

type redirectHandler struct {
	router  *chain.Router
	md      metadata
	options handler.Options
}

func NewHandler(opts ...handler.Option) handler.Handler {
	options := handler.Options{}
	for _, opt := range opts {
		opt(&options)
	}

	return &redirectHandler{
		options: options,
	}
}

func (h *redirectHandler) Init(md md.Metadata) (err error) {
	if err = h.parseMetadata(md); err != nil {
		return
	}

	h.router = h.options.Router
	if h.router == nil {
		h.router = (&chain.Router{}).WithLogger(h.options.Logger)
	}

	return
}

func (h *redirectHandler) Handle(ctx context.Context, conn net.Conn) error {
	defer conn.Close()

	start := time.Now()
	log := h.options.Logger.WithFields(map[string]any{
		"remote": conn.RemoteAddr().String(),
		"local":  conn.LocalAddr().String(),
	})

	log.Infof("%s <> %s", conn.RemoteAddr(), conn.LocalAddr())
	defer func() {
		log.WithFields(map[string]any{
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
			log.Error(err)
			return err
		}
	}

	log = log.WithFields(map[string]any{
		"dst": fmt.Sprintf("%s/%s", dstAddr, network),
	})

	log.Infof("%s >> %s", conn.RemoteAddr(), dstAddr)

	if h.options.Bypass != nil && h.options.Bypass.Contains(dstAddr.String()) {
		log.Info("bypass: ", dstAddr)
		return nil
	}

	cc, err := h.router.Dial(ctx, network, dstAddr.String())
	if err != nil {
		log.Error(err)
		return err
	}
	defer cc.Close()

	t := time.Now()
	log.Infof("%s <-> %s", conn.RemoteAddr(), dstAddr)
	netpkg.Transport(conn, cc)
	log.WithFields(map[string]any{
		"duration": time.Since(t),
	}).Infof("%s >-< %s", conn.RemoteAddr(), dstAddr)

	return nil
}
