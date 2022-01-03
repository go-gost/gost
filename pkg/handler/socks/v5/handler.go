package v5

import (
	"context"
	"net"
	"time"

	"github.com/go-gost/gosocks5"
	"github.com/go-gost/gost/pkg/auth"
	"github.com/go-gost/gost/pkg/bypass"
	"github.com/go-gost/gost/pkg/chain"
	"github.com/go-gost/gost/pkg/common/util/socks"
	"github.com/go-gost/gost/pkg/handler"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
)

func init() {
	registry.RegisterHandler("socks5", NewHandler)
	registry.RegisterHandler("socks", NewHandler)
}

type socks5Handler struct {
	selector      gosocks5.Selector
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

	return &socks5Handler{
		bypass: options.Bypass,
		router: options.Router,
		logger: options.Logger,
	}
}

func (h *socks5Handler) Init(md md.Metadata) (err error) {
	if err = h.parseMetadata(md); err != nil {
		return
	}

	h.selector = &serverSelector{
		Authenticator: h.authenticator,
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

	address := req.Addr.String()

	switch req.Cmd {
	case gosocks5.CmdConnect:
		h.handleConnect(ctx, conn, "tcp", address)
	case gosocks5.CmdBind:
		h.handleBind(ctx, conn, "tcp", address)
	case socks.CmdMuxBind:
		h.handleMuxBind(ctx, conn, "tcp", address)
	case gosocks5.CmdUdp:
		h.handleUDP(ctx, conn)
	case socks.CmdUDPTun:
		h.handleUDPTun(ctx, conn, "udp", address)
	default:
		h.logger.Errorf("unknown cmd: %d", req.Cmd)
		resp := gosocks5.NewReply(gosocks5.CmdUnsupported, nil)
		resp.Write(conn)
		h.logger.Debug(resp)
		return
	}
}
