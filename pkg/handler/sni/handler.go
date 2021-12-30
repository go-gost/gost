package sni

import (
	"bytes"
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
	"github.com/go-gost/gost/pkg/common/bufpool"
	"github.com/go-gost/gost/pkg/handler"
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
	bypass      bypass.Bypass
	router      *chain.Router
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
		router: (&chain.Router{}).
			WithLogger(options.Logger).
			WithResolver(options.Resolver),
		logger: log,
	}

	if f := registry.GetHandler("http"); f != nil {
		v := append(opts,
			handler.LoggerOption(log.WithFields(map[string]interface{}{"type": "http"})))
		h.httpHandler = f(v...)
	}

	return h
}

func (h *sniHandler) Init(md md.Metadata) (err error) {
	if err = h.parseMetadata(md); err != nil {
		return
	}
	if h.httpHandler != nil {
		if md != nil {
			md.Set("sni", true)
		}
		if err = h.httpHandler.Init(md); err != nil {
			return
		}
	}

	h.router.WithRetry(h.md.retryCount)

	return nil
}

// WithChain implements chain.Chainable interface
func (h *sniHandler) WithChain(chain *chain.Chain) {
	h.router.WithChain(chain)
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

	var hdr [dissector.RecordHeaderLen]byte
	if _, err := io.ReadFull(conn, hdr[:]); err != nil {
		h.logger.Error(err)
		return
	}

	if hdr[0] != dissector.Handshake {
		// We assume it is an HTTP request
		conn = &cacheConn{
			Conn: conn,
			buf:  hdr[:],
		}

		if h.httpHandler != nil {
			h.httpHandler.Handle(ctx, conn)
		}
		return
	}

	length := binary.BigEndian.Uint16(hdr[3:5])

	buf := bufpool.Get(int(length) + dissector.RecordHeaderLen)
	defer bufpool.Put(buf)
	if _, err := io.ReadFull(conn, buf[dissector.RecordHeaderLen:]); err != nil {
		h.logger.Error(err)
		return
	}
	copy(buf, hdr[:])

	buf, host, err := h.decodeHost(bytes.NewReader(buf))
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

	cc, err := h.router.Dial(ctx, "tcp", target)
	if err != nil {
		return
	}
	defer cc.Close()

	if _, err := cc.Write(buf); err != nil {
		h.logger.Error(err)
		return
	}

	t := time.Now()
	h.logger.Infof("%s <-> %s", conn.RemoteAddr(), target)
	handler.Transport(conn, cc)
	h.logger.
		WithFields(map[string]interface{}{
			"duration": time.Since(t),
		}).
		Infof("%s >-< %s", conn.RemoteAddr(), target)
}

func (h *sniHandler) decodeHost(r io.Reader) (opaque []byte, host string, err error) {
	record, err := dissector.ReadRecord(r)
	if err != nil {
		return
	}
	clientHello := dissector.ClientHelloMsg{}
	if err = clientHello.Decode(record.Opaque); err != nil {
		return
	}

	var extensions []dissector.Extension
	for _, ext := range clientHello.Extensions {
		if ext.Type() == 0xFFFE {
			b, _ := ext.Encode()
			if v, err := h.decodeServerName(string(b)); err == nil {
				host = v
			}
			continue
		}
		extensions = append(extensions, ext)
	}
	clientHello.Extensions = extensions

	for _, ext := range clientHello.Extensions {
		if ext.Type() == dissector.ExtServerName {
			snExtension := ext.(*dissector.ServerNameExtension)
			if host == "" {
				host = snExtension.Name
			} else {
				snExtension.Name = host
			}
			break
		}
	}

	record.Opaque, err = clientHello.Encode()
	if err != nil {
		return
	}

	buf := &bytes.Buffer{}
	if _, err = record.WriteTo(buf); err != nil {
		return
	}
	opaque = buf.Bytes()
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
