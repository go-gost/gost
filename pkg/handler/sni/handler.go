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

	"github.com/go-gost/gost/pkg/chain"
	"github.com/go-gost/gost/pkg/common/bufpool"
	netpkg "github.com/go-gost/gost/pkg/common/net"
	"github.com/go-gost/gost/pkg/handler"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
	dissector "github.com/go-gost/tls-dissector"
)

func init() {
	registry.HandlerRegistry().Register("sni", NewHandler)
}

type sniHandler struct {
	httpHandler handler.Handler
	router      *chain.Router
	md          metadata
	options     handler.Options
}

func NewHandler(opts ...handler.Option) handler.Handler {
	options := handler.Options{}
	for _, opt := range opts {
		opt(&options)
	}

	h := &sniHandler{
		options: options,
	}

	if f := registry.HandlerRegistry().Get("http"); f != nil {
		v := append(opts,
			handler.LoggerOption(h.options.Logger.WithFields(map[string]any{"type": "http"})))
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

	h.router = h.options.Router
	if h.router == nil {
		h.router = (&chain.Router{}).WithLogger(h.options.Logger)
	}

	return nil
}

func (h *sniHandler) Handle(ctx context.Context, conn net.Conn) error {
	defer conn.Close()

	start := time.Now()
	log := h.options.Logger.WithFields(map[string]any{
		"remote": conn.RemoteAddr().String(),
		"local":  conn.LocalAddr().String(),
	})

	log.Infof("%s <> %s", conn.RemoteAddr(), conn.LocalAddr())
	defer func() {
		log.WithFields(map[string]any{
			"duration": time.Since(start),
		}).Infof("%s >< %s", conn.RemoteAddr(), conn.LocalAddr())
	}()

	var hdr [dissector.RecordHeaderLen]byte
	if _, err := io.ReadFull(conn, hdr[:]); err != nil {
		log.Error(err)
		return err
	}

	if hdr[0] != dissector.Handshake {
		// We assume it is an HTTP request
		conn = &cacheConn{
			Conn: conn,
			buf:  hdr[:],
		}

		if h.httpHandler != nil {
			return h.httpHandler.Handle(ctx, conn)
		}
		return nil
	}

	length := binary.BigEndian.Uint16(hdr[3:5])

	buf := bufpool.Get(int(length) + dissector.RecordHeaderLen)
	defer bufpool.Put(buf)
	if _, err := io.ReadFull(conn, (*buf)[dissector.RecordHeaderLen:]); err != nil {
		log.Error(err)
		return err
	}
	copy(*buf, hdr[:])

	opaque, host, err := h.decodeHost(bytes.NewReader(*buf))
	if err != nil {
		log.Error(err)
		return err
	}
	target := net.JoinHostPort(host, "443")

	log = log.WithFields(map[string]any{
		"dst": target,
	})
	log.Infof("%s >> %s", conn.RemoteAddr(), target)

	if h.options.Bypass != nil && h.options.Bypass.Contains(target) {
		log.Info("bypass: ", target)
		return nil
	}

	cc, err := h.router.Dial(ctx, "tcp", target)
	if err != nil {
		log.Error(err)
		return err
	}
	defer cc.Close()

	if _, err := cc.Write(opaque); err != nil {
		log.Error(err)
		return err
	}

	t := time.Now()
	log.Infof("%s <-> %s", conn.RemoteAddr(), target)
	netpkg.Transport(conn, cc)
	log.WithFields(map[string]any{
		"duration": time.Since(t),
	}).Infof("%s >-< %s", conn.RemoteAddr(), target)

	return nil
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
