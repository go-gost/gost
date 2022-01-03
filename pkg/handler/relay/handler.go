package relay

import (
	"context"
	"net"
	"strconv"
	"time"

	"github.com/go-gost/gost/pkg/auth"
	"github.com/go-gost/gost/pkg/bypass"
	"github.com/go-gost/gost/pkg/chain"
	"github.com/go-gost/gost/pkg/handler"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
	"github.com/go-gost/relay"
)

func init() {
	registry.RegisterHandler("relay", NewHandler)
}

type relayHandler struct {
	group         *chain.NodeGroup
	bypass        bypass.Bypass
	router        *chain.Router
	authenticator auth.Authenticator
	logger        logger.Logger
	md            metadata
}

func NewHandler(opts ...handler.Option) handler.Handler {
	options := &handler.Options{}
	for _, opt := range opts {
		opt(options)
	}

	return &relayHandler{
		bypass: options.Bypass,
		router: options.Router,
		logger: options.Logger,
	}
}

func (h *relayHandler) Init(md md.Metadata) (err error) {
	if err := h.parseMetadata(md); err != nil {
		return err
	}

	return nil
}

// Forward implements handler.Forwarder.
func (h *relayHandler) Forward(group *chain.NodeGroup) {
	h.group = group
}

func (h *relayHandler) Handle(ctx context.Context, conn net.Conn) {
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

	if h.md.readTimeout > 0 {
		conn.SetReadDeadline(time.Now().Add(h.md.readTimeout))
	}

	req := relay.Request{}
	if _, err := req.ReadFrom(conn); err != nil {
		h.logger.Error(err)
		return
	}

	conn.SetReadDeadline(time.Time{})

	if req.Version != relay.Version1 {
		h.logger.Error("bad version")
		return
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
		h.logger = h.logger.WithFields(map[string]interface{}{"user": user})
	}

	resp := relay.Response{
		Version: relay.Version1,
		Status:  relay.StatusOK,
	}
	if h.authenticator != nil && !h.authenticator.Authenticate(user, pass) {
		resp.Status = relay.StatusUnauthorized
		resp.WriteTo(conn)
		h.logger.Error("unauthorized")
		return
	}

	network := "tcp"
	if (req.Flags & relay.FUDP) == relay.FUDP {
		network = "udp"
	}

	if h.group != nil {
		if address != "" {
			resp.Status = relay.StatusForbidden
			resp.WriteTo(conn)
			h.logger.Error("forward mode, connect is forbidden")
			return
		}
		// forward mode
		h.handleForward(ctx, conn, network)
		return
	}

	switch req.Flags & relay.CmdMask {
	case 0, relay.CONNECT:
		h.handleConnect(ctx, conn, network, address)
	case relay.BIND:
		h.handleBind(ctx, conn, network, address)
	}
}
