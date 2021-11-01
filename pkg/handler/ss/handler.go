package ss

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"time"

	"github.com/go-gost/gosocks5"
	"github.com/go-gost/gost/pkg/bypass"
	"github.com/go-gost/gost/pkg/chain"
	"github.com/go-gost/gost/pkg/handler"
	"github.com/go-gost/gost/pkg/internal/utils"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
)

func init() {
	registry.RegisterHandler("ss", NewHandler)
}

type ssHandler struct {
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

	return &ssHandler{
		chain:  options.Chain,
		bypass: options.Bypass,
		logger: options.Logger,
	}
}

func (h *ssHandler) Init(md md.Metadata) (err error) {
	return h.parseMetadata(md)
}

func (h *ssHandler) Handle(ctx context.Context, conn net.Conn) {
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

	sc := conn
	if h.md.cipher != nil {
		sc = utils.ShadowConn(h.md.cipher.StreamConn(conn), nil)
	}

	if h.md.readTimeout > 0 {
		sc.SetReadDeadline(time.Now().Add(h.md.readTimeout))
	}

	addr := &gosocks5.Addr{}
	_, err := addr.ReadFrom(sc)
	if err != nil {
		h.logger.Error(err)
		h.discard(conn)
		return
	}

	sc.SetReadDeadline(time.Time{})

	h.logger = h.logger.WithFields(map[string]interface{}{
		"dst": addr.String(),
	})

	h.logger.Infof("%s > %s", conn.RemoteAddr(), addr)

	if h.bypass != nil && h.bypass.Contains(addr.String()) {
		h.logger.Info("bypass: ", addr.String())
		return
	}

	cc, err := h.dial(ctx, addr.String())
	if err != nil {
		return
	}
	defer cc.Close()

	h.logger.Infof("%s <> %s", conn.RemoteAddr(), addr)
	handler.Transport(sc, cc)
	h.logger.Infof("%s >< %s", conn.RemoteAddr(), addr)
}

func (h *ssHandler) discard(conn net.Conn) {
	io.Copy(ioutil.Discard, conn)
}

func (h *ssHandler) parseMetadata(md md.Metadata) (err error) {
	h.md.cipher, err = utils.ShadowCipher(
		md.GetString(method),
		md.GetString(password),
		md.GetString(key),
	)
	if err != nil {
		return
	}

	h.md.readTimeout = md.GetDuration(readTimeout)
	h.md.retryCount = md.GetInt(retryCount)
	return
}

func (h *ssHandler) dial(ctx context.Context, addr string) (conn net.Conn, err error) {
	count := h.md.retryCount + 1
	if count <= 0 {
		count = 1
	}

	for i := 0; i < count; i++ {
		route := h.chain.GetRouteFor(addr)

		if h.logger.IsLevelEnabled(logger.DebugLevel) {
			buf := bytes.Buffer{}
			for _, node := range route.Path() {
				fmt.Fprintf(&buf, "%s@%s > ", node.Name(), node.Addr())
			}
			fmt.Fprintf(&buf, "%s", addr)
			h.logger.Debugf("route(retry=%d): %s", i, buf.String())
		}

		conn, err = route.Dial(ctx, "tcp", addr)
		if err == nil {
			break
		}
		h.logger.Errorf("route(retry=%d): %s", i, err)
	}

	return
}
