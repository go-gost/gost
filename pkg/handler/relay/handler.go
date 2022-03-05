package relay

import (
	"context"
	"errors"
	"net"
	"strconv"
	"time"

	"github.com/go-gost/gost/pkg/chain"
	"github.com/go-gost/gost/pkg/handler"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
	"github.com/go-gost/relay"
)

var (
	ErrBadVersion = errors.New("relay: bad version")
	ErrUnknownCmd = errors.New("relay: unknown command")
)

func init() {
	registry.HandlerRegistry().Register("relay", NewHandler)
}

type relayHandler struct {
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

	return &relayHandler{
		options: options,
	}
}

func (h *relayHandler) Init(md md.Metadata) (err error) {
	if err := h.parseMetadata(md); err != nil {
		return err
	}

	h.router = h.options.Router
	if h.router == nil {
		h.router = (&chain.Router{}).WithLogger(h.options.Logger)
	}

	return nil
}

// Forward implements handler.Forwarder.
func (h *relayHandler) Forward(group *chain.NodeGroup) {
	h.group = group
}

func (h *relayHandler) Handle(ctx context.Context, conn net.Conn) error {
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

	if h.md.readTimeout > 0 {
		conn.SetReadDeadline(time.Now().Add(h.md.readTimeout))
	}

	req := relay.Request{}
	if _, err := req.ReadFrom(conn); err != nil {
		log.Error(err)
		return err
	}

	conn.SetReadDeadline(time.Time{})

	if req.Version != relay.Version1 {
		err := ErrBadVersion
		log.Error(err)
		return err
	}

	var user, pass string
	var address string
	for _, f := range req.Features {
		if f.Type() == relay.FeatureUserAuth {
			feature := f.(*relay.UserAuthFeature)
			user, pass = feature.Username, feature.Password
		}
		if f.Type() == relay.FeatureAddr {
			feature := f.(*relay.AddrFeature)
			address = net.JoinHostPort(feature.Host, strconv.Itoa(int(feature.Port)))
		}
	}

	if user != "" {
		log = log.WithFields(map[string]any{"user": user})
	}

	resp := relay.Response{
		Version: relay.Version1,
		Status:  relay.StatusOK,
	}
	if h.options.Auther != nil && !h.options.Auther.Authenticate(user, pass) {
		resp.Status = relay.StatusUnauthorized
		log.Error("unauthorized")
		_, err := resp.WriteTo(conn)
		return err
	}

	network := "tcp"
	if (req.Flags & relay.FUDP) == relay.FUDP {
		network = "udp"
	}

	if h.group != nil {
		if address != "" {
			resp.Status = relay.StatusForbidden
			log.Error("forward mode, connect is forbidden")
			_, err := resp.WriteTo(conn)
			return err
		}
		// forward mode
		return h.handleForward(ctx, conn, network, log)
	}

	switch req.Flags & relay.CmdMask {
	case 0, relay.CONNECT:
		return h.handleConnect(ctx, conn, network, address, log)
	case relay.BIND:
		return h.handleBind(ctx, conn, network, address, log)
	}
	return ErrUnknownCmd
}
