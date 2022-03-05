package local

import (
	"context"
	"errors"
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
	registry.HandlerRegistry().Register("tcp", NewHandler)
	registry.HandlerRegistry().Register("udp", NewHandler)
	registry.HandlerRegistry().Register("forward", NewHandler)
}

type forwardHandler struct {
	group   *chain.NodeGroup
	router  *chain.Router
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

	if h.group == nil {
		// dummy node used by relay connector.
		h.group = chain.NewNodeGroup(&chain.Node{Name: "dummy", Addr: ":0"})
	}

	h.router = h.options.Router
	if h.router == nil {
		h.router = (&chain.Router{}).WithLogger(h.options.Logger)
	}

	return
}

// Forward implements handler.Forwarder.
func (h *forwardHandler) Forward(group *chain.NodeGroup) {
	h.group = group
}

func (h *forwardHandler) Handle(ctx context.Context, conn net.Conn) error {
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

	target := h.group.Next()
	if target == nil {
		err := errors.New("target not available")
		log.Error(err)
		return err
	}

	network := "tcp"
	if _, ok := conn.(net.PacketConn); ok {
		network = "udp"
	}

	log = log.WithFields(map[string]any{
		"dst": fmt.Sprintf("%s/%s", target.Addr, network),
	})

	log.Infof("%s >> %s", conn.RemoteAddr(), target.Addr)

	cc, err := h.router.Dial(ctx, network, target.Addr)
	if err != nil {
		log.Error(err)
		// TODO: the router itself may be failed due to the failed node in the router,
		// the dead marker may be a wrong operation.
		target.Marker.Mark()
		return err
	}
	defer cc.Close()
	target.Marker.Reset()

	t := time.Now()
	log.Infof("%s <-> %s", conn.RemoteAddr(), target.Addr)
	netpkg.Transport(conn, cc)
	log.WithFields(map[string]any{
		"duration": time.Since(t),
	}).Infof("%s >-< %s", conn.RemoteAddr(), target.Addr)

	return nil
}
