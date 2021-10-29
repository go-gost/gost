package ss

import (
	"context"
	"net"
	"time"

	"github.com/go-gost/gost/pkg/components/handler"
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

func (h *Handler) Init(md handler.Metadata) (err error) {
	h.md, err = h.parseMetadata(md)
	if err != nil {
		return
	}
	return nil
}

func (h *Handler) Handle(ctx context.Context, conn net.Conn) {
	defer conn.Close()
}

func (h *Handler) parseMetadata(md handler.Metadata) (m metadata, err error) {
	m.cipher, err = h.initCipher(md[method], md[password], md[key])
	if err != nil {
		return
	}
	if v, ok := md[readTimeout]; ok {
		m.readTimeout, _ = time.ParseDuration(v)
	}
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
