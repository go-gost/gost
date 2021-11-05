package ssu

import (
	"context"
	"net"
	"time"

	"github.com/go-gost/gost/pkg/bypass"
	"github.com/go-gost/gost/pkg/chain"
	"github.com/go-gost/gost/pkg/handler"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
	"github.com/shadowsocks/go-shadowsocks2/core"
	ss "github.com/shadowsocks/shadowsocks-go/shadowsocks"
)

func init() {
	registry.RegisterHandler("ssu", NewHandler)
}

type ssuHandler struct {
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

	return &ssuHandler{
		chain:  options.Chain,
		bypass: options.Bypass,
		logger: options.Logger,
	}
}

func (h *ssuHandler) Init(md md.Metadata) (err error) {
	return h.parseMetadata(md)
}

func (h *ssuHandler) Handle(ctx context.Context, conn net.Conn) {
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

}

func (h *ssuHandler) parseMetadata(md md.Metadata) (err error) {
	h.md.cipher, err = h.initCipher(
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

func (h *ssuHandler) initCipher(method, password string, key string) (core.Cipher, error) {
	if method == "" && password == "" {
		return nil, nil
	}

	c, _ := ss.NewCipher(method, password)
	if c != nil {
		return &shadowCipher{cipher: c}, nil
	}

	return core.PickCipher(method, []byte(key), password)
}

type shadowCipher struct {
	cipher *ss.Cipher
}

func (c *shadowCipher) StreamConn(conn net.Conn) net.Conn {
	return ss.NewConn(conn, c.cipher.Copy())
}

func (c *shadowCipher) PacketConn(conn net.PacketConn) net.PacketConn {
	return ss.NewSecurePacketConn(conn, c.cipher.Copy())
}
