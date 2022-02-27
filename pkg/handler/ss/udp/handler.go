package ss

import (
	"context"
	"net"
	"time"

	"github.com/go-gost/gost/pkg/chain"
	"github.com/go-gost/gost/pkg/common/bufpool"
	"github.com/go-gost/gost/pkg/common/util/socks"
	"github.com/go-gost/gost/pkg/common/util/ss"
	"github.com/go-gost/gost/pkg/handler"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
	"github.com/shadowsocks/go-shadowsocks2/core"
)

func init() {
	registry.HandlerRegistry().Register("ssu", NewHandler)
}

type ssuHandler struct {
	cipher  core.Cipher
	router  *chain.Router
	md      metadata
	options handler.Options
}

func NewHandler(opts ...handler.Option) handler.Handler {
	options := handler.Options{}
	for _, opt := range opts {
		opt(&options)
	}

	return &ssuHandler{
		options: options,
	}
}

func (h *ssuHandler) Init(md md.Metadata) (err error) {
	if err = h.parseMetadata(md); err != nil {
		return
	}

	if h.options.Auth != nil {
		method := h.options.Auth.Username()
		password, _ := h.options.Auth.Password()
		h.cipher, err = ss.ShadowCipher(method, password, h.md.key)
		if err != nil {
			return
		}
	}

	h.router = &chain.Router{
		Retries:  h.options.Retries,
		Chain:    h.options.Chain,
		Resolver: h.options.Resolver,
		Hosts:    h.options.Hosts,
		Logger:   h.options.Logger,
	}

	return
}

func (h *ssuHandler) Handle(ctx context.Context, conn net.Conn) {
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

	pc, ok := conn.(net.PacketConn)
	if ok {
		if h.cipher != nil {
			pc = h.cipher.PacketConn(pc)
		}
		// standard UDP relay.
		pc = ss.UDPServerConn(pc, conn.RemoteAddr(), h.md.bufferSize)
	} else {
		if h.cipher != nil {
			conn = ss.ShadowConn(h.cipher.StreamConn(conn), nil)
		}
		// UDP over TCP
		pc = socks.UDPTunServerConn(conn)
	}

	// obtain a udp connection
	c, err := h.router.Dial(ctx, "udp", "") // UDP association
	if err != nil {
		log.Error(err)
		return
	}
	defer c.Close()

	cc, ok := c.(net.PacketConn)
	if !ok {
		log.Errorf("wrong connection type")
		return
	}

	t := time.Now()
	log.Infof("%s <-> %s", conn.LocalAddr(), cc.LocalAddr())
	h.relayPacket(pc, cc, log)
	log.WithFields(map[string]any{"duration": time.Since(t)}).
		Infof("%s >-< %s", conn.LocalAddr(), cc.LocalAddr())
}

func (h *ssuHandler) relayPacket(pc1, pc2 net.PacketConn, log logger.Logger) (err error) {
	bufSize := h.md.bufferSize
	errc := make(chan error, 2)

	go func() {
		for {
			err := func() error {
				b := bufpool.Get(bufSize)
				defer bufpool.Put(b)

				n, addr, err := pc1.ReadFrom(*b)
				if err != nil {
					return err
				}

				if h.options.Bypass != nil && h.options.Bypass.Contains(addr.String()) {
					log.Warn("bypass: ", addr)
					return nil
				}

				if _, err = pc2.WriteTo((*b)[:n], addr); err != nil {
					return err
				}

				log.Debugf("%s >>> %s data: %d",
					pc2.LocalAddr(), addr, n)
				return nil
			}()

			if err != nil {
				errc <- err
				return
			}
		}
	}()

	go func() {
		for {
			err := func() error {
				b := bufpool.Get(bufSize)
				defer bufpool.Put(b)

				n, raddr, err := pc2.ReadFrom(*b)
				if err != nil {
					return err
				}

				if h.options.Bypass != nil && h.options.Bypass.Contains(raddr.String()) {
					log.Warn("bypass: ", raddr)
					return nil
				}

				if _, err = pc1.WriteTo((*b)[:n], raddr); err != nil {
					return err
				}

				log.Debugf("%s <<< %s data: %d",
					pc2.LocalAddr(), raddr, n)
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
