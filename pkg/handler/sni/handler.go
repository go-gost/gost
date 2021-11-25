package sni

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"io"
	"net"
	"time"

	"github.com/go-gost/gost/pkg/bypass"
	"github.com/go-gost/gost/pkg/chain"
	"github.com/go-gost/gost/pkg/handler"
	http_handler "github.com/go-gost/gost/pkg/handler/http"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
	dissector "github.com/go-gost/tls-dissector"
)

func init() {
	registry.RegisterHandler("sni", NewHandler)
}

type sniHandler struct {
	httpHandler handler.Handler
	chain       *chain.Chain
	bypass      bypass.Bypass
	logger      logger.Logger
	md          metadata
}

func NewHandler(opts ...handler.Option) handler.Handler {
	options := &handler.Options{}
	for _, opt := range opts {
		opt(options)
	}

	log := options.Logger
	if log == nil {
		log = logger.Default()
	}

	h := &sniHandler{
		bypass: options.Bypass,
		logger: log,
	}

	v := append(opts,
		handler.LoggerOption(log.WithFields(map[string]interface{}{"type": "http"})))
	h.httpHandler = http_handler.NewHandler(v...)

	return h
}

func (h *sniHandler) Init(md md.Metadata) (err error) {
	if err = h.parseMetadata(md); err != nil {
		return
	}
	if err = h.httpHandler.Init(md); err != nil {
		return
	}

	return nil
}

// WithChain implements chain.Chainable interface
func (h *sniHandler) WithChain(chain *chain.Chain) {
	h.chain = chain
}

func (h *sniHandler) Handle(ctx context.Context, conn net.Conn) {
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

	br := bufio.NewReader(conn)
	hdr, err := br.Peek(dissector.RecordHeaderLen)
	if err != nil {
		h.logger.Error(err)
		return
	}

	conn = handler.NewBufferReaderConn(conn, br)

	if hdr[0] != dissector.Handshake {
		// We assume it is an HTTP request
		h.httpHandler.Handle(ctx, conn)
		return
	}

	host, err := h.decodeHost(conn)
	if err != nil {
		h.logger.Error(err)
		return
	}
	target := net.JoinHostPort(host, "443")

	h.logger = h.logger.WithFields(map[string]interface{}{
		"dst": target,
	})
	h.logger.Infof("%s >> %s", conn.RemoteAddr(), target)

	if h.bypass != nil && h.bypass.Contains(target) {
		h.logger.Info("bypass: ", target)
		return
	}

	r := (&chain.Router{}).
		WithChain(h.chain).
		WithRetry(h.md.retryCount).
		WithLogger(h.logger)
	cc, err := r.Dial(ctx, "tcp", target)
	if err != nil {
		return
	}
	defer cc.Close()

	t := time.Now()
	h.logger.Infof("%s <-> %s", conn.RemoteAddr(), target)
	handler.Transport(conn, cc)
	h.logger.
		WithFields(map[string]interface{}{
			"duration": time.Since(t),
		}).
		Infof("%s >-< %s", conn.RemoteAddr(), target)
}

func (h *sniHandler) decodeHost(r io.Reader) (host string, err error) {
	record, err := dissector.ReadRecord(r)
	if err != nil {
		return
	}
	clientHello := &dissector.ClientHelloMsg{}
	if err = clientHello.Decode(record.Opaque); err != nil {
		return
	}

	for _, ext := range clientHello.Extensions {
		if ext.Type() == 0xFFFE {
			b, _ := ext.Encode()
			return h.decodeServerName(string(b))
		}

		if ext.Type() == dissector.ExtServerName {
			snExtension := ext.(*dissector.ServerNameExtension)
			host = snExtension.Name
		}
	}
	return
}

func (h *sniHandler) decodeServerName(s string) (string, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}
	if len(b) < 4 {
		return "", errors.New("invalid name")
	}
	v, err := base64.RawURLEncoding.DecodeString(string(b[4:]))
	if err != nil {
		return "", err
	}
	if crc32.ChecksumIEEE(v) != binary.BigEndian.Uint32(b[:4]) {
		return "", errors.New("invalid name")
	}
	return string(v), nil
}
