package tap

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"

	"github.com/go-gost/gost/pkg/chain"
	"github.com/go-gost/gost/pkg/common/bufpool"
	"github.com/go-gost/gost/pkg/common/util/ss"
	"github.com/go-gost/gost/pkg/handler"
	tap_util "github.com/go-gost/gost/pkg/internal/util/tap"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
	"github.com/shadowsocks/go-shadowsocks2/core"
	"github.com/shadowsocks/go-shadowsocks2/shadowaead"
	"github.com/songgao/water/waterutil"
)

func init() {
	registry.HandlerRegistry().Register("tap", NewHandler)
}

type tapHandler struct {
	group   *chain.NodeGroup
	routes  sync.Map
	exit    chan struct{}
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

	return &tapHandler{
		exit:    make(chan struct{}, 1),
		options: options,
	}
}

func (h *tapHandler) Init(md md.Metadata) (err error) {
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

	h.router = h.options.Router
	if h.router == nil {
		h.router = (&chain.Router{}).WithLogger(h.options.Logger)
	}

	return
}

// Forward implements handler.Forwarder.
func (h *tapHandler) Forward(group *chain.NodeGroup) {
	h.group = group
}

func (h *tapHandler) Handle(ctx context.Context, conn net.Conn) error {
	defer os.Exit(0)
	defer conn.Close()

	log := h.options.Logger
	cc, ok := conn.(*tap_util.Conn)
	if !ok || cc.Config() == nil {
		err := errors.New("tap: wrong connection type")
		log.Error(err)
		return err
	}

	start := time.Now()
	log = log.WithFields(map[string]any{
		"remote": conn.RemoteAddr().String(),
		"local":  conn.LocalAddr().String(),
	})

	log.Infof("%s <> %s", conn.RemoteAddr(), conn.LocalAddr())
	defer func() {
		log.WithFields(map[string]any{
			"duration": time.Since(start),
		}).Infof("%s >< %s", conn.RemoteAddr(), conn.LocalAddr())
	}()

	network := "udp"
	var raddr net.Addr
	var err error

	target := h.group.Next()
	if target != nil {
		raddr, err = net.ResolveUDPAddr(network, target.Addr)
		if err != nil {
			log.Error(err)
			return err
		}
		log = log.WithFields(map[string]any{
			"dst": fmt.Sprintf("%s/%s", raddr.String(), raddr.Network()),
		})
		log.Infof("%s >> %s", conn.RemoteAddr(), target.Addr)
	}

	h.handleLoop(ctx, conn, raddr, cc.Config(), log)
	return nil
}

func (h *tapHandler) handleLoop(ctx context.Context, conn net.Conn, addr net.Addr, config *tap_util.Config, log logger.Logger) {
	var tempDelay time.Duration
	for {
		err := func() error {
			var err error
			var pc net.PacketConn

			if addr != nil {
				cc, err := h.router.Dial(ctx, addr.Network(), addr.String())
				if err != nil {
					return err
				}

				var ok bool
				pc, ok = cc.(net.PacketConn)
				if !ok {
					return errors.New("invalid connection")
				}
			} else {
				laddr, _ := net.ResolveUDPAddr("udp", conn.LocalAddr().String())
				pc, err = net.ListenUDP("udp", laddr)
			}
			if err != nil {
				return err
			}

			if h.cipher != nil {
				pc = h.cipher.PacketConn(pc)
			}

			return h.transport(conn, pc, addr, log)
		}()
		if err != nil {
			log.Error(err)
		}

		select {
		case <-h.exit:
			return
		default:
		}

		if err != nil {
			if tempDelay == 0 {
				tempDelay = 1000 * time.Millisecond
			} else {
				tempDelay *= 2
			}
			if max := 6 * time.Second; tempDelay > max {
				tempDelay = max
			}
			time.Sleep(tempDelay)
			continue
		}
		tempDelay = 0
	}

}

func (h *tapHandler) transport(tap net.Conn, conn net.PacketConn, raddr net.Addr, log logger.Logger) error {
	errc := make(chan error, 1)

	go func() {
		for {
			err := func() error {
				b := bufpool.Get(h.md.bufferSize)
				defer bufpool.Put(b)

				n, err := tap.Read(*b)
				if err != nil {
					select {
					case h.exit <- struct{}{}:
					default:
					}
					return err
				}

				src := waterutil.MACSource((*b)[:n])
				dst := waterutil.MACDestination((*b)[:n])
				eType := etherType(waterutil.MACEthertype((*b)[:n]))

				log.Debugf("%s >> %s %s %d", src, dst, eType, n)

				// client side, deliver frame directly.
				if raddr != nil {
					_, err := conn.WriteTo((*b)[:n], raddr)
					return err
				}

				// server side, broadcast.
				if waterutil.IsBroadcast(dst) {
					go h.routes.Range(func(k, v any) bool {
						conn.WriteTo((*b)[:n], v.(net.Addr))
						return true
					})
					return nil
				}

				var addr net.Addr
				if v, ok := h.routes.Load(hwAddrToTapRouteKey(dst)); ok {
					addr = v.(net.Addr)
				}
				if addr == nil {
					log.Warnf("no route for %s -> %s %s %d", src, dst, eType, n)
					return nil
				}

				if _, err := conn.WriteTo((*b)[:n], addr); err != nil {
					return err
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
		for {
			err := func() error {
				b := bufpool.Get(h.md.bufferSize)
				defer bufpool.Put(b)

				n, addr, err := conn.ReadFrom(*b)
				if err != nil &&
					err != shadowaead.ErrShortPacket {
					return err
				}

				src := waterutil.MACSource((*b)[:n])
				dst := waterutil.MACDestination((*b)[:n])
				eType := etherType(waterutil.MACEthertype((*b)[:n]))

				log.Debugf("%s >> %s %s %d", src, dst, eType, n)

				// client side, deliver frame to tap device.
				if raddr != nil {
					_, err := tap.Write((*b)[:n])
					return err
				}

				// server side, record route.
				rkey := hwAddrToTapRouteKey(src)
				if actual, loaded := h.routes.LoadOrStore(rkey, addr); loaded {
					if actual.(net.Addr).String() != addr.String() {
						log.Debugf("update route: %s -> %s (old %s)",
							src, addr, actual.(net.Addr))
						h.routes.Store(rkey, addr)
					}
				} else {
					log.Debugf("new route: %s -> %s", src, addr)
				}

				if waterutil.IsBroadcast(dst) {
					go h.routes.Range(func(k, v any) bool {
						if k.(tapRouteKey) != rkey {
							conn.WriteTo((*b)[:n], v.(net.Addr))
						}
						return true
					})
				}

				if v, ok := h.routes.Load(hwAddrToTapRouteKey(dst)); ok {
					log.Debugf("find route: %s -> %s", dst, v)
					_, err := conn.WriteTo((*b)[:n], v.(net.Addr))
					return err
				}

				if _, err := tap.Write((*b)[:n]); err != nil {
					select {
					case h.exit <- struct{}{}:
					default:
					}
					return err
				}
				return nil
			}()

			if err != nil {
				errc <- err
				return
			}
		}
	}()

	err := <-errc
	if err != nil && err == io.EOF {
		err = nil
	}
	return err
}

var mEtherTypes = map[waterutil.Ethertype]string{
	waterutil.IPv4: "ip",
	waterutil.ARP:  "arp",
	waterutil.RARP: "rarp",
	waterutil.IPv6: "ip6",
}

func etherType(et waterutil.Ethertype) string {
	if s, ok := mEtherTypes[et]; ok {
		return s
	}
	return fmt.Sprintf("unknown(%v)", et)
}

type tapRouteKey [6]byte

func hwAddrToTapRouteKey(addr net.HardwareAddr) (key tapRouteKey) {
	copy(key[:], addr)
	return
}
