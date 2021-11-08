package ssu

import (
	"bytes"
	"context"
	"net"
	"time"

	"github.com/go-gost/gosocks5"
	"github.com/go-gost/gost/pkg/bypass"
	"github.com/go-gost/gost/pkg/chain"
	"github.com/go-gost/gost/pkg/handler"
	"github.com/go-gost/gost/pkg/internal/bufpool"
	"github.com/go-gost/gost/pkg/internal/utils/ss"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
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

	// obtain a udp connection
	r := (&handler.Router{}).
		WithChain(h.chain).
		WithRetry(h.md.retryCount).
		WithLogger(h.logger)
	c, err := r.Dial(ctx, "udp", "")
	if err != nil {
		h.logger.Error(err)
		return
	}

	cc, ok := c.(net.PacketConn)
	if !ok {
		h.logger.Errorf("%s: not a packet connection")
		return
	}
	defer cc.Close()

	h.logger = h.logger.WithFields(map[string]interface{}{
		"bind": cc.LocalAddr().String(),
	})
	h.logger.Infof("bind on %s OK", cc.LocalAddr().String())

	pc, ok := conn.(net.PacketConn)
	if ok {
		if h.md.cipher != nil {
			pc = h.md.cipher.PacketConn(pc)
		}

		t := time.Now()
		h.logger.Infof("%s <-> %s", conn.RemoteAddr(), cc.LocalAddr())
		h.relayPacket(pc, cc)
		h.logger.
			WithFields(map[string]interface{}{"duration": time.Since(t)}).
			Infof("%s >-< %s", conn.RemoteAddr(), cc.LocalAddr())
		return
	}

	if h.md.cipher != nil {
		conn = ss.ShadowConn(h.md.cipher.StreamConn(conn), nil)
	}

	t := time.Now()
	h.logger.Infof("%s <-> %s", conn.RemoteAddr(), cc.LocalAddr())
	h.tunnelUDP(conn, cc)
	h.logger.
		WithFields(map[string]interface{}{"duration": time.Since(t)}).
		Infof("%s >-< %s", conn.RemoteAddr(), cc.LocalAddr())
}

func (h *ssuHandler) relayPacket(pc1, pc2 net.PacketConn) (err error) {
	bufSize := h.md.bufferSize

	errc := make(chan error, 2)
	var clientAddr net.Addr

	go func() {
		b := bufpool.Get(bufSize)
		defer bufpool.Put(b)

		for {
			err := func() error {
				n, addr, err := pc1.ReadFrom(b)
				if err != nil {
					return err
				}
				if clientAddr == nil {
					clientAddr = addr
				}

				rb := bytes.NewBuffer(b[:n])
				saddr := gosocks5.Addr{}
				if _, err := saddr.ReadFrom(rb); err != nil {
					return err
				}
				taddr, err := net.ResolveUDPAddr("udp", saddr.String())
				if err != nil {
					return err
				}

				if h.bypass != nil && h.bypass.Contains(taddr.String()) {
					h.logger.Warn("bypass: ", taddr)
					return nil
				}

				if _, err = pc2.WriteTo(rb.Bytes(), taddr); err != nil {
					return err
				}

				if h.logger.IsLevelEnabled(logger.DebugLevel) {
					h.logger.Debugf("%s >>> %s: %v, data: %d",
						addr, taddr, saddr.String(), rb.Len())
				}
				return nil
			}()

			if err != nil {
				errc <- err
				return
			}
		}
	}()

	go func() {
		b := bufpool.Get(bufSize)
		defer bufpool.Put(b)

		const dataPos = 259

		for {
			err := func() error {
				n, raddr, err := pc2.ReadFrom(b[dataPos:])
				if err != nil {
					return err
				}
				if clientAddr == nil {
					return nil
				}

				if h.bypass != nil && h.bypass.Contains(raddr.String()) {
					h.logger.Warn("bypass: ", raddr)
					return nil
				}

				socksAddr, _ := gosocks5.NewAddr(raddr.String())
				if socksAddr == nil {
					socksAddr = &gosocks5.Addr{}
				}
				addrLen := socksAddr.Length()
				socksAddr.Encode(b[dataPos-addrLen : dataPos])

				if _, err = pc1.WriteTo(b[dataPos-addrLen:dataPos+n], clientAddr); err != nil {
					return err
				}

				if h.logger.IsLevelEnabled(logger.DebugLevel) {
					h.logger.Debugf("%s <<< %s: %v data: %d",
						clientAddr, raddr, b[dataPos-addrLen:dataPos], n)
				}
				return nil
			}()

			if err != nil {
				errc <- err
				return
			}
		}
	}()

	return <-errc
}

func (h *ssuHandler) tunnelUDP(tunnel net.Conn, c net.PacketConn) (err error) {
	bufSize := h.md.bufferSize
	errc := make(chan error, 2)

	go func() {
		b := bufpool.Get(bufSize)
		defer bufpool.Put(b)

		const dataPos = 262

		for {
			addr := gosocks5.Addr{}
			header := gosocks5.UDPHeader{
				Addr: &addr,
			}

			data := b[dataPos:]
			dgram := gosocks5.UDPDatagram{
				Header: &header,
				Data:   data,
			}
			_, err := dgram.ReadFrom(tunnel)
			if err != nil {
				errc <- err
				return
			}
			// NOTE: the dgram.Data may be reallocated if the provided buffer is too short,
			// we drop it for simplicity. As this occurs, you should enlarge the buffer size.
			if len(dgram.Data) > len(data) {
				h.logger.Warnf("buffer too short, dropped")
				continue
			}

			raddr, err := net.ResolveUDPAddr("udp", addr.String())
			if err != nil {
				continue // drop silently
			}
			if h.bypass != nil && h.bypass.Contains(raddr.String()) {
				h.logger.Warn("bypass: ", raddr.String())
				continue // bypass
			}

			if _, err := c.WriteTo(dgram.Data, raddr); err != nil {
				errc <- err
				return
			}

			if h.logger.IsLevelEnabled(logger.DebugLevel) {
				h.logger.Debugf("%s >>> %s: %v data: %d",
					tunnel.RemoteAddr(), raddr, header.String(), len(dgram.Data))
			}
		}
	}()

	go func() {
		b := bufpool.Get(bufSize)
		defer bufpool.Put(b)

		for {
			n, raddr, err := c.ReadFrom(b)
			if err != nil {
				errc <- err
				return
			}

			if h.bypass != nil && h.bypass.Contains(raddr.String()) {
				h.logger.Warn("bypass: ", raddr.String())
				continue // bypass
			}

			addr, _ := gosocks5.NewAddr(raddr.String())
			if addr == nil {
				addr = &gosocks5.Addr{}
			}
			header := gosocks5.UDPHeader{
				Rsv:  uint16(n),
				Addr: addr,
			}
			dgram := gosocks5.UDPDatagram{
				Header: &header,
				Data:   b[:n],
			}

			if _, err := dgram.WriteTo(tunnel); err != nil {
				errc <- err
				return
			}
			if h.logger.IsLevelEnabled(logger.DebugLevel) {
				h.logger.Debugf("%s <<< %s: %v data: %d",
					tunnel.RemoteAddr(), raddr, header.String(), len(dgram.Data))
			}
		}
	}()

	return <-errc
}

func (h *ssuHandler) parseMetadata(md md.Metadata) (err error) {
	h.md.cipher, err = ss.ShadowCipher(
		md.GetString(method),
		md.GetString(password),
		md.GetString(key),
	)
	if err != nil {
		return
	}

	h.md.readTimeout = md.GetDuration(readTimeout)
	h.md.retryCount = md.GetInt(retryCount)

	h.md.bufferSize = md.GetInt(bufferSize)
	if h.md.bufferSize > 0 {
		if h.md.bufferSize < 512 {
			h.md.bufferSize = 512 // min buffer size
		}
		if h.md.bufferSize > 65*1024 {
			h.md.bufferSize = 65 * 1024 // max buffer size
		}
	} else {
		h.md.bufferSize = 4096 // default buffer size
	}

	return
}
