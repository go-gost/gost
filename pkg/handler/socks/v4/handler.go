package v4

import (
	"context"
	"net"
	"time"

	"github.com/go-gost/gosocks4"
	"github.com/go-gost/gost/pkg/auth"
	"github.com/go-gost/gost/pkg/bypass"
	"github.com/go-gost/gost/pkg/chain"
	"github.com/go-gost/gost/pkg/handler"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
)

func init() {
	registry.RegisterHandler("socks4", NewHandler)
	registry.RegisterHandler("socks4a", NewHandler)
}

type socks4Handler struct {
	chain  *chain.Chain
	bypass bypass.Bypass
	logger logger.Logger
	md     metadata
}

func NewHandler(opts ...handler.Option) handler.Handler {
	options := &handler.Options{}
	for _, opt := range opts {
		opt(options)
	}

	return &socks4Handler{
		chain:  options.Chain,
		bypass: options.Bypass,
		logger: options.Logger,
	}
}

func (h *socks4Handler) Init(md md.Metadata) (err error) {
	return h.parseMetadata(md)
}

func (h *socks4Handler) Handle(ctx context.Context, conn net.Conn) {
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

	req, err := gosocks4.ReadRequest(conn)
	if err != nil {
		h.logger.Error(err)
		return
	}
	h.logger.Debug(req)

	conn.SetReadDeadline(time.Time{})

	if h.md.authenticator != nil &&
		!h.md.authenticator.Authenticate(string(req.Userid), "") {
		resp := gosocks4.NewReply(gosocks4.RejectedUserid, nil)
		resp.Write(conn)
		h.logger.Debug(resp)
		return
	}

	switch req.Cmd {
	case gosocks4.CmdConnect:
		h.handleConnect(ctx, conn, req)
	case gosocks4.CmdBind:
		h.handleBind(ctx, conn, req)
	default:
		h.logger.Errorf("unknown cmd: %d", req.Cmd)
	}
}

func (h *socks4Handler) handleConnect(ctx context.Context, conn net.Conn, req *gosocks4.Request) {
	addr := req.Addr.String()

	h.logger = h.logger.WithFields(map[string]interface{}{
		"dst": addr,
	})
	h.logger.Infof("%s >> %s", conn.RemoteAddr(), addr)

	if h.bypass != nil && h.bypass.Contains(addr) {
		resp := gosocks4.NewReply(gosocks4.Rejected, nil)
		resp.Write(conn)
		h.logger.Debug(resp)
		h.logger.Info("bypass: ", addr)
		return
	}

	r := (&handler.Router{}).
		WithChain(h.chain).
		WithRetry(h.md.retryCount).
		WithLogger(h.logger)
	cc, err := r.Dial(ctx, "tcp", addr)
	if err != nil {
		resp := gosocks4.NewReply(gosocks4.Failed, nil)
		resp.Write(conn)
		h.logger.Debug(resp)
		return
	}

	defer cc.Close()

	resp := gosocks4.NewReply(gosocks4.Granted, nil)
	if err := resp.Write(conn); err != nil {
		h.logger.Error(err)
		return
	}
	h.logger.Debug(resp)

	t := time.Now()
	h.logger.Infof("%s <-> %s", conn.RemoteAddr(), addr)
	handler.Transport(conn, cc)
	h.logger.
		WithFields(map[string]interface{}{
			"duration": time.Since(t),
		}).
		Infof("%s >-< %s", conn.RemoteAddr(), addr)
}

func (h *socks4Handler) handleBind(ctx context.Context, conn net.Conn, req *gosocks4.Request) {
	// TODO: bind
}

func (h *socks4Handler) parseMetadata(md md.Metadata) (err error) {
	if v, _ := md.Get(authsKey).([]interface{}); len(v) > 0 {
		authenticator := auth.NewLocalAuthenticator(nil)
		for _, auth := range v {
			if v, _ := auth.(string); v != "" {
				authenticator.Add(v, "")
			}
		}
		h.md.authenticator = authenticator
	}

	h.md.readTimeout = md.GetDuration(readTimeout)
	h.md.retryCount = md.GetInt(retryCount)
	return
}
