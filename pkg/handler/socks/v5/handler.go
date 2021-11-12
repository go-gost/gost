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
	h.logger.Debug(req)
	conn.SetReadDeadline(time.Time{})

	switch req.Cmd {
	case gosocks5.CmdConnect:
		h.handleConnect(ctx, conn, req.Addr.String())
	case gosocks5.CmdBind:
		if h.md.enableBind {
			h.handleBind(ctx, conn, req)
		} else {
			h.logger.Error("BIND is diabled")
		}
	case socks.CmdMuxBind:
		if h.md.enableBind {
			h.handleMuxBind(ctx, conn, req)
		} else {
			h.logger.Error("BIND is diabled")
		}
	case gosocks5.CmdUdp:
		if h.md.enableUDP {
			h.handleUDP(ctx, conn, req)
		} else {
			h.logger.Error("UDP relay is diabled")
		}
	case socks.CmdUDPTun:
		if h.md.enableUDP {
			h.handleUDPTun(ctx, conn, req)
		} else {
			h.logger.Error("UDP relay is diabled")
		}
	default:
		h.logger.Errorf("unknown cmd: %d", req.Cmd)
		resp := gosocks5.NewReply(gosocks5.CmdUnsupported, nil)
		resp.Write(conn)
		h.logger.Debug(resp)
		return
	}
}
