package ss

import (
	"bytes"
	"context"
	"net"
	"time"

	"github.com/go-gost/gosocks5"
	"github.com/go-gost/gost/pkg/components/handler"
	md "github.com/go-gost/gost/pkg/components/metadata"
	"github.com/go-gost/gost/pkg/logger"
	"github.com/go-gost/gost/pkg/registry"
	"github.com/shadowsocks/go-shadowsocks2/core"
	ss "github.com/shadowsocks/shadowsocks-go/shadowsocks"
)

func init() {
	registry.RegisterHandler("ss", NewHandler)
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

	if h.md.cipher != nil {
		conn = &shadowConn{
			Conn: h.md.cipher.StreamConn(conn),
		}
	}

	if h.md.readTimeout > 0 {
		conn.SetReadDeadline(time.Now().Add(h.md.readTimeout))
	}

	addr := &gosocks5.Addr{}
	_, err := addr.ReadFrom(conn)
	if err != nil {
		h.logger.Error(err)
		return
	}

	conn.SetReadDeadline(time.Time{})

	host := addr.String()
	cc, err := net.Dial("tcp", host)
	if err != nil {
		return
	}
	defer cc.Close()

	handler.Transport(conn, cc)
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

// Due to in/out byte length is inconsistent of the shadowsocks.Conn.Write,
// we wrap around it to make io.Copy happy.
type shadowConn struct {
	net.Conn
	wbuf bytes.Buffer
}

func (c *shadowConn) Write(b []byte) (n int, err error) {
	n = len(b) // force byte length consistent
	if c.wbuf.Len() > 0 {
		c.wbuf.Write(b) // append the data to the cached header
		_, err = c.Conn.Write(c.wbuf.Bytes())
		c.wbuf.Reset()
		return
	}
	_, err = c.Conn.Write(b)
	return
}
