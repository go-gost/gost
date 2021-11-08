package v5

import (
	"context"
	"net"
	"time"

	"github.com/go-gost/gosocks5"
	"github.com/go-gost/gost/pkg/bypass"
	"github.com/go-gost/gost/pkg/chain"
	"github.com/go-gost/gost/pkg/handler"
	"github.com/go-gost/gost/pkg/internal/utils/socks"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
)

func init() {
	registry.RegisterHandler("socks5", NewHandler)
	registry.RegisterHandler("socks", NewHandler)
}

type socks5Handler struct {
	selector gosocks5.Selector
	chain    *chain.Chain
	bypass   bypass.Bypass
	logger   logger.Logger
	md       metadata
}

func NewHandler(opts ...handler.Option) handler.Handler {
	options := &handler.Options{}
	for _, opt := range opts {
		opt(options)
	}

	return &socks5Handler{
		chain:  options.Chain,
		bypass: options.Bypass,
		logger: options.Logger,
	}
}

func (h *socks5Handler) Init(md md.Metadata) (err error) {
	if err = h.parseMetadata(md); err != nil {
		return
	}

	h.selector = &serverSelector{
		Authenticator: h.md.authenticator,
		TLSConfig:     h.md.tlsConfig,
		logger:        h.logger,
		noTLS:         h.md.noTLS,
	}

	return
}

func (h *socks5Handler) Handle(ctx context.Context, conn net.Conn) {
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

	conn = gosocks5.ServerConn(conn, h.selector)
	req, err := gosocks5.ReadRequest(conn)
	if err != nil {
		h.logger.Error(err)
		return
	}
	conn.SetReadDeadline(time.Time{})

	if h.logger.IsLevelEnabled(logger.DebugLevel) {
		h.logger.Debug(req)
	}

	switch req.Cmd {
	case gosocks5.CmdConnect:
		h.handleConnect(ctx, conn, req.Addr.String())
	case gosocks5.CmdBind:
		h.handleBind(ctx, conn, req)
	case socks.CmdMuxBind:
		h.handleMuxBind(ctx, conn, req)
	case gosocks5.CmdUdp:
		h.handleUDP(ctx, conn, req)
	case socks.CmdUDPTun:
		h.handleUDPTun(ctx, conn, req)
	default:
		h.logger.Errorf("unknown cmd: %d", req.Cmd)
		resp := gosocks5.NewReply(gosocks5.CmdUnsupported, nil)
		resp.Write(conn)
		if h.logger.IsLevelEnabled(logger.DebugLevel) {
			h.logger.Debug(resp)
		}
		return
	}
}
