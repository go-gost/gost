package remote

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/go-gost/gost/pkg/chain"
	"github.com/go-gost/gost/pkg/handler"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
)

func init() {
	registry.RegisterHandler("rtcp", NewHandler)
	registry.RegisterHandler("rudp", NewHandler)
}

type forwardHandler struct {
	group   *chain.NodeGroup
	router  *chain.Router
	logger  logger.Logger
	md      metadata
	options handler.Options
}

func NewHandler(opts ...handler.Option) handler.Handler {
	options := handler.Options{}
	for _, opt := range opts {
		opt(&options)
	}

	return &forwardHandler{
		options: options,
	}
}

func (h *forwardHandler) Init(md md.Metadata) (err error) {
	if err = h.parseMetadata(md); err != nil {
		return
	}

	h.router = &chain.Router{
		Retries:  h.options.Retries,
		Chain:    h.options.Chain,
		Resolver: h.options.Resolver,
		Hosts:    h.options.Hosts,
		Logger:   h.options.Logger,
	}
	h.logger = h.options.Logger

	return
}

// Forward implements handler.Forwarder.
func (h *forwardHandler) Forward(group *chain.NodeGroup) {
	h.group = group
}

func (h *forwardHandler) Handle(ctx context.Context, conn net.Conn) {
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

	target := h.group.Next()
	if target == nil {
		h.logger.Error("no target available")
		return
	}

	network := "tcp"
	if _, ok := conn.(net.PacketConn); ok {
		network = "udp"
	}

	h.logger = h.logger.WithFields(map[string]interface{}{
		"dst": fmt.Sprintf("%s/%s", target.Addr(), network),
	})

	h.logger.Infof("%s >> %s", conn.RemoteAddr(), target.Addr())

	cc, err := h.router.Dial(ctx, network, target.Addr())
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
