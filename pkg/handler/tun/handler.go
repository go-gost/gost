package tun

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"

	"github.com/go-gost/gost/pkg/bypass"
	"github.com/go-gost/gost/pkg/chain"
	"github.com/go-gost/gost/pkg/common/bufpool"
	"github.com/go-gost/gost/pkg/handler"
	tun_util "github.com/go-gost/gost/pkg/internal/util/tun"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
	"github.com/shadowsocks/go-shadowsocks2/shadowaead"
	"github.com/songgao/water/waterutil"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

func init() {
	registry.RegisterHandler("tun", NewHandler)
}

type tunHandler struct {
	group  *chain.NodeGroup
	bypass bypass.Bypass
	routes sync.Map
	exit   chan struct{}
	router *chain.Router
	logger logger.Logger
	md     metadata
}

func NewHandler(opts ...handler.Option) handler.Handler {
	options := &handler.Options{}
	for _, opt := range opts {
		opt(options)
	}

	return &tunHandler{
		bypass: options.Bypass,
		router: (&chain.Router{}).
			WithLogger(options.Logger).
			WithResolver(options.Resolver),
		logger: options.Logger,
		exit:   make(chan struct{}, 1),
	}
}

func (h *tunHandler) Init(md md.Metadata) (err error) {
	if err := h.parseMetadata(md); err != nil {
		return err
	}

	h.router.WithRetry(h.md.retryCount)

	return nil
}

// implements chain.Chainable interface
func (h *tunHandler) WithChain(chain *chain.Chain) {
	h.router.WithChain(chain)
}

// Forward implements handler.Forwarder.
func (h *tunHandler) Forward(group *chain.NodeGroup) {
	h.group = group
}

func (h *tunHandler) Handle(ctx context.Context, conn net.Conn) {
	defer os.Exit(0)
	defer conn.Close()

	cc, ok := conn.(*tun_util.Conn)
	if !ok || cc.Config() == nil {
		h.logger.Error("invalid connection")
		return
	}

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

	network := "udp"
	var raddr net.Addr
	var err error

	target := h.group.Next()
	if target != nil {
		raddr, err = net.ResolveUDPAddr(network, target.Addr())
		if err != nil {
			h.logger.Error(err)
			return
		}
		h.logger = h.logger.WithFields(map[string]interface{}{
			"dst": fmt.Sprintf("%s/%s", raddr.String(), raddr.Network()),
		})
		h.logger.Infof("%s >> %s", conn.RemoteAddr(), target.Addr())
	}

	h.handleLoop(ctx, conn, raddr, cc.Config())
}

func (h *tunHandler) handleLoop(ctx context.Context, conn net.Conn, addr net.Addr, config *tun_util.Config) {
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
					return errors.New("invalid connnection")
				}
			} else {
				laddr, _ := net.ResolveUDPAddr("udp", conn.LocalAddr().String())
				pc, err = net.ListenUDP("udp", laddr)
			}
			if err != nil {
				return err
			}

			if h.md.cipher != nil {
				pc = h.md.cipher.PacketConn(pc)
			}

			return h.transport(conn, pc, addr)
		}()
		if err != nil {
			h.logger.Error(err)
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

func (h *tunHandler) transport(tun net.Conn, conn net.PacketConn, raddr net.Addr) error {
	errc := make(chan error, 1)

	go func() {
		for {
			err := func() error {
				b := bufpool.Get(h.md.bufferSize)
				defer bufpool.Put(b)

				n, err := tun.Read(b)
				if err != nil {
					select {
					case h.exit <- struct{}{}:
					default:
					}
					return err
				}

				var src, dst net.IP
				if waterutil.IsIPv4(b[:n]) {
					header, err := ipv4.ParseHeader(b[:n])
					if err != nil {
						h.logger.Error(err)
						return nil
					}
					h.logger.Debugf("%s >> %s %-4s %d/%-4d %-4x %d",
						header.Src, header.Dst, ipProtocol(waterutil.IPv4Protocol(b[:n])),
						header.Len, header.TotalLen, header.ID, header.Flags)

					src, dst = header.Src, header.Dst
				} else if waterutil.IsIPv6(b[:n]) {
					header, err := ipv6.ParseHeader(b[:n])
					if err != nil {
						h.logger.Warn(err)
						return nil
					}
					h.logger.Debugf("%s >> %s %s %d %d",
						header.Src, header.Dst,
						ipProtocol(waterutil.IPProtocol(header.NextHeader)),
						header.PayloadLen, header.TrafficClass)

					src, dst = header.Src, header.Dst
				} else {
					h.logger.Warn("unknown packet, discarded")
					return nil
				}

				// client side, deliver packet directly.
				if raddr != nil {
					_, err := conn.WriteTo(b[:n], raddr)
					return err
				}

				addr := h.findRouteFor(dst)
				if addr == nil {
					h.logger.Warnf("no route for %s -> %s", src, dst)
					return nil
				}

				h.logger.Debugf("find route: %s -> %s", dst, addr)

				if _, err := conn.WriteTo(b[:n], addr); err != nil {
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

				n, addr, err := conn.ReadFrom(b)
				if err != nil &&
					err != shadowaead.ErrShortPacket {
					return err
				}

				var src, dst net.IP
				if waterutil.IsIPv4(b[:n]) {
					header, err := ipv4.ParseHeader(b[:n])
					if err != nil {
						h.logger.Warn(err)
						return nil
					}

					h.logger.Debugf("%s >> %s %-4s %d/%-4d %-4x %d",
						header.Src, header.Dst, ipProtocol(waterutil.IPv4Protocol(b[:n])),
						header.Len, header.TotalLen, header.ID, header.Flags)

					src, dst = header.Src, header.Dst
				} else if waterutil.IsIPv6(b[:n]) {
					header, err := ipv6.ParseHeader(b[:n])
					if err != nil {
						h.logger.Warn(err)
						return nil
					}

					h.logger.Debugf("%s > %s %s %d %d",
						header.Src, header.Dst,
						ipProtocol(waterutil.IPProtocol(header.NextHeader)),
						header.PayloadLen, header.TrafficClass)

					src, dst = header.Src, header.Dst
				} else {
					h.logger.Warn("unknown packet, discarded")
					return nil
				}

				// client side, deliver packet to tun device.
				if raddr != nil {
					_, err := tun.Write(b[:n])
					return err
				}

				rkey := ipToTunRouteKey(src)
				if actual, loaded := h.routes.LoadOrStore(rkey, addr); loaded {
					if actual.(net.Addr).String() != addr.String() {
						h.logger.Debugf("update route: %s -> %s (old %s)",
							src, addr, actual.(net.Addr))
						h.routes.Store(rkey, addr)
					}
				} else {
					h.logger.Warnf("no route for %s -> %s", src, addr)
				}

				if addr := h.findRouteFor(dst); addr != nil {
					h.logger.Debugf("find route: %s -> %s", dst, addr)

					_, err := conn.WriteTo(b[:n], addr)
					return err
				}

				if _, err := tun.Write(b[:n]); err != nil {
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

func (h *tunHandler) findRouteFor(dst net.IP, routes ...tun_util.Route) net.Addr {
	if v, ok := h.routes.Load(ipToTunRouteKey(dst)); ok {
		return v.(net.Addr)
	}
	for _, route := range routes {
		if route.Net.Contains(dst) && route.Gateway != nil {
			if v, ok := h.routes.Load(ipToTunRouteKey(route.Gateway)); ok {
				return v.(net.Addr)
			}
		}
	}
	return nil
}

var mIPProts = map[waterutil.IPProtocol]string{
	waterutil.HOPOPT:     "HOPOPT",
	waterutil.ICMP:       "ICMP",
	waterutil.IGMP:       "IGMP",
	waterutil.GGP:        "GGP",
	waterutil.TCP:        "TCP",
	waterutil.UDP:        "UDP",
	waterutil.IPv6_Route: "IPv6-Route",
	waterutil.IPv6_Frag:  "IPv6-Frag",
	waterutil.IPv6_ICMP:  "IPv6-ICMP",
}

func ipProtocol(p waterutil.IPProtocol) string {
	if v, ok := mIPProts[p]; ok {
		return v
	}
	return fmt.Sprintf("unknown(%d)", p)
}

type tunRouteKey [16]byte

func ipToTunRouteKey(ip net.IP) (key tunRouteKey) {
	copy(key[:], ip.To16())
	return
}
