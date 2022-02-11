package v5

import (
	"context"
	"net"
	"time"

	"github.com/go-gost/gosocks5"
	"github.com/go-gost/gost/pkg/chain"
	"github.com/go-gost/gost/pkg/common/util/socks"
	"github.com/go-gost/gost/pkg/handler"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
)

func init() {
	registry.RegisterHandler("socks5", NewHandler)
	registry.RegisterHandler("socks", NewHandler)
}

type socks5Handler struct {
	selector gosocks5.Selector
	router   *chain.Router
	md       metadata
	options  handler.Options
}

func NewHandler(opts ...handler.Option) handler.Handler {
	options := handler.Options{}
	for _, opt := range opts {
		opt(&options)
	}

	return &socks5Handler{
		options: options,
	}
}

func (h *socks5Handler) Init(md md.Metadata) (err error) {
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

	h.selector = &serverSelector{
		Authenticator: h.options.Auther,
		TLSConfig:     h.options.TLSConfig,
		logger:        h.options.Logger,
		noTLS:         h.md.noTLS,
	}

	return
}

func (h *socks5Handler) Handle(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	start := time.Now()

	log := h.options.Logger.WithFields(map[string]interface{}{
		"remote": conn.RemoteAddr().String(),
		"local":  conn.LocalAddr().String(),
	})

	log.Infof("%s <> %s", conn.RemoteAddr(), conn.LocalAddr())
	defer func() {
		log.WithFields(map[string]interface{}{
			"duration": time.Since(start),
		}).Infof("%s >< %s", conn.RemoteAddr(), conn.LocalAddr())
	}()

	if h.md.readTimeout > 0 {
		conn.SetReadDeadline(time.Now().Add(h.md.readTimeout))
	}

	conn = gosocks5.ServerConn(conn, h.selector)
	req, err := gosocks5.ReadRequest(conn)
	if err != nil {
		log.Error(err)
		return
	}
	log.Debug(req)
	conn.SetReadDeadline(time.Time{})

	address := req.Addr.String()

	switch req.Cmd {
	case gosocks5.CmdConnect:
		h.handleConnect(ctx, conn, "tcp", address, log)
	case gosocks5.CmdBind:
		h.handleBind(ctx, conn, "tcp", address, log)
	case socks.CmdMuxBind:
		h.handleMuxBind(ctx, conn, "tcp", address, log)
	case gosocks5.CmdUdp:
		h.handleUDP(ctx, conn, log)
	case socks.CmdUDPTun:
		h.handleUDPTun(ctx, conn, "udp", address, log)
	default:
		log.Errorf("unknown cmd: %d", req.Cmd)
		resp := gosocks5.NewReply(gosocks5.CmdUnsupported, nil)
		resp.Write(conn)
		log.Debug(resp)
		return
	}
}
