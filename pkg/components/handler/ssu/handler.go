package ss

import (
	"context"
	"net"

	"github.com/go-gost/gost/pkg/components/handler"
	md "github.com/go-gost/gost/pkg/components/metadata"
	"github.com/go-gost/gost/pkg/logger"
	"github.com/go-gost/gost/pkg/registry"
	"github.com/shadowsocks/go-shadowsocks2/core"
	ss "github.com/shadowsocks/shadowsocks-go/shadowsocks"
)

func init() {
	registry.RegisterHandler("ssu", NewHandler)
}

type Handler struct {
	logger logger.Logger
	md     metadata
}

func NewHandler(opts ...handler.Option) handler.Handler {
	options := &handler.Options{}
	for _, opt := range opts {
		opt(options)
	}

	return &Handler{
		logger: options.Logger,
	}
}

func (h *Handler) Init(md md.Metadata) (err error) {
	return h.parseMetadata(md)
}

func (h *Handler) Handle(ctx context.Context, conn net.Conn) {
	defer conn.Close()
}

func (h *Handler) parseMetadata(md md.Metadata) (err error) {
	h.md.cipher, err = h.initCipher(
		md.GetString(method),
		md.GetString(password),
		md.GetString(key),
	)
	if err != nil {
		return
	}

	h.md.readTimeout = md.GetDuration(readTimeout)

	return
}

func (h *Handler) initCipher(method, password string, key string) (core.Cipher, error) {
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
