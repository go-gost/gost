package ss

import (
	"context"
	"net"
	"time"

	"github.com/go-gost/gost/pkg/bypass"
	"github.com/go-gost/gost/pkg/chain"
	"github.com/go-gost/gost/pkg/common/bufpool"
	"github.com/go-gost/gost/pkg/common/util/socks"
	"github.com/go-gost/gost/pkg/common/util/ss"
	"github.com/go-gost/gost/pkg/handler"
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
		bypass: options.Bypass,
		logger: options.Logger,
	}
}

func (h *ssuHandler) Init(md md.Metadata) (err error) {
	return h.parseMetadata(md)
}

// WithChain implements chain.Chainable interface
func (h *ssuHandler) WithChain(chain *chain.Chain) {
	h.chain = chain
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

	pc, ok := conn.(net.PacketConn)
	if ok {
		if h.md.cipher != nil {
			pc = h.md.cipher.PacketConn(pc)
		}
		// standard UDP relay.
		pc = ss.UDPServerConn(pc, conn.RemoteAddr(), h.md.bufferSize)
	} else {
		if h.md.cipher != nil {
			conn = ss.ShadowConn(h.md.cipher.StreamConn(conn), nil)
		}
		// UDP over TCP
		pc = socks.UDPTunServerConn(conn)
	}

	// obtain a udp connection
	r := (&chain.Router{}).
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

	t := time.Now()
	h.logger.Infof("%s <-> %s", conn.RemoteAddr(), cc.LocalAddr())
	h.relayPacket(pc, cc)
	h.logger.
		WithFields(map[string]interface{}{"duration": time.Since(t)}).
		Infof("%s >-< %s", conn.RemoteAddr(), cc.LocalAddr())
}

func (h *ssuHandler) relayPacket(pc1, pc2 net.PacketConn) (err error) {
	bufSize := h.md.bufferSize
	errc := make(chan error, 2)

	go func() {
		for {
			err := func() error {
				b := bufpool.Get(bufSize)
				defer bufpool.Put(b)

				n, addr, err := pc1.ReadFrom(b)
				if err != nil {
					return err
				}

				if h.bypass != nil && h.bypass.Contains(addr.String()) {
					h.logger.Warn("bypass: ", addr)
					return nil
				}

				if _, err = pc2.WriteTo(b[:n], addr); err != nil {
					return err
				}

				h.logger.Debugf("%s >>> %s data: %d",
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

				n, raddr, err := pc2.ReadFrom(b)
				if err != nil {
					return err
				}

				if h.bypass != nil && h.bypass.Contains(raddr.String()) {
					h.logger.Warn("bypass: ", raddr)
					return nil
				}

				if _, err = pc1.WriteTo(b[:n], raddr); err != nil {
					return err
				}

				h.logger.Debugf("%s <<< %s data: %d",
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
