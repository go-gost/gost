package ssh

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/go-gost/gost/pkg/chain"
	"github.com/go-gost/gost/pkg/handler"
	sshd_util "github.com/go-gost/gost/pkg/internal/util/sshd"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
	"golang.org/x/crypto/ssh"
)

// Applicable SSH Request types for Port Forwarding - RFC 4254 7.X
const (
	ForwardedTCPReturnRequest = "forwarded-tcpip" // RFC 4254 7.2
)

func init() {
	registry.HandlerRegistry().Register("sshd", NewHandler)
}

type forwardHandler struct {
	router  *chain.Router
	md      metadata
	options handler.Options
}

func NewHandler(opts ...handler.Option) handler.Handler {
	options := handler.Options{}
	for _, opt := range opts {
		opt(&options)
	}

	return &forwardHandler{
		options: options,
	}
}

func (h *forwardHandler) Init(md md.Metadata) (err error) {
	if err = h.parseMetadata(md); err != nil {
		return
	}

	h.router = &chain.Router{
		Retries:  h.options.Retries,
		Chain:    h.options.Chain,
		Resolver: h.options.Resolver,
		Hosts:    h.options.Hosts,
		Logger:   h.options.Logger,
	}

	return nil
}

func (h *forwardHandler) Handle(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	log := h.options.Logger.WithFields(map[string]any{
		"remote": conn.RemoteAddr().String(),
		"local":  conn.LocalAddr().String(),
	})

	switch cc := conn.(type) {
	case *sshd_util.DirectForwardConn:
		h.handleDirectForward(ctx, cc, log)
	case *sshd_util.RemoteForwardConn:
		h.handleRemoteForward(ctx, cc, log)
	default:
		log.Error("wrong connection type")
		return
	}
}

func (h *forwardHandler) handleDirectForward(ctx context.Context, conn *sshd_util.DirectForwardConn, log logger.Logger) {
	targetAddr := conn.DstAddr()

	log = log.WithFields(map[string]any{
		"dst": fmt.Sprintf("%s/%s", targetAddr, "tcp"),
		"cmd": "connect",
	})

	log.Infof("%s >> %s", conn.RemoteAddr(), targetAddr)

	if h.options.Bypass != nil && h.options.Bypass.Contains(targetAddr) {
		log.Infof("bypass %s", targetAddr)
		return
	}

	cc, err := h.router.Dial(ctx, "tcp", targetAddr)
	if err != nil {
		return
	}
	defer cc.Close()

	t := time.Now()
	log.Infof("%s <-> %s", cc.LocalAddr(), targetAddr)
	handler.Transport(conn, cc)
	log.WithFields(map[string]any{
		"duration": time.Since(t),
	}).Infof("%s >-< %s", cc.LocalAddr(), targetAddr)
}

func (h *forwardHandler) handleRemoteForward(ctx context.Context, conn *sshd_util.RemoteForwardConn, log logger.Logger) {
	req := conn.Request()

	t := tcpipForward{}
	ssh.Unmarshal(req.Payload, &t)

	network := "tcp"
	addr := net.JoinHostPort(t.Host, strconv.Itoa(int(t.Port)))

	log = log.WithFields(map[string]any{
		"dst": fmt.Sprintf("%s/%s", addr, network),
		"cmd": "bind",
	})

	log.Infof("%s >> %s", conn.RemoteAddr(), addr)

	// tie to the client connection
	ln, err := net.Listen(network, addr)
	if err != nil {
		log.Error(err)
		req.Reply(false, nil)
		return
	}
	defer ln.Close()

	log = log.WithFields(map[string]any{
		"bind": fmt.Sprintf("%s/%s", ln.Addr(), ln.Addr().Network()),
	})
	log.Debugf("bind on %s OK", ln.Addr())

	err = func() error {
		if t.Port == 0 && req.WantReply { // Client sent port 0. let them know which port is actually being used
			_, port, err := getHostPortFromAddr(ln.Addr())
			if err != nil {
				return err
			}
			var b [4]byte
			binary.BigEndian.PutUint32(b[:], uint32(port))
			t.Port = uint32(port)
			return req.Reply(true, b[:])
		}
		return req.Reply(true, nil)
	}()
	if err != nil {
		log.Error(err)
		return
	}

	sshConn := conn.Conn()

	go func() {
		for {
			cc, err := ln.Accept()
			if err != nil { // Unable to accept new connection - listener is likely closed
				return
			}

			go func(conn net.Conn) {
				defer conn.Close()

				log := log.WithFields(map[string]any{
					"local":  conn.LocalAddr().String(),
					"remote": conn.RemoteAddr().String(),
				})

				p := directForward{}
				var err error

				var portnum int
				p.Host1 = t.Host
				p.Port1 = t.Port
				p.Host2, portnum, err = getHostPortFromAddr(conn.RemoteAddr())
				if err != nil {
					return
				}

				p.Port2 = uint32(portnum)
				ch, reqs, err := sshConn.OpenChannel(ForwardedTCPReturnRequest, ssh.Marshal(p))
				if err != nil {
					log.Error("open forwarded channel: ", err)
					return
				}
				defer ch.Close()
				go ssh.DiscardRequests(reqs)

				t := time.Now()
				log.Infof("%s <-> %s", conn.LocalAddr(), conn.RemoteAddr())
				handler.Transport(ch, conn)
				log.WithFields(map[string]any{
					"duration": time.Since(t),
				}).Infof("%s >-< %s", conn.LocalAddr(), conn.RemoteAddr())
			}(cc)
		}
	}()

	tm := time.Now()
	log.Infof("%s <-> %s", conn.RemoteAddr(), addr)
	<-conn.Done()
	log.WithFields(map[string]any{
		"duration": time.Since(tm),
	}).Infof("%s >-< %s", conn.RemoteAddr(), addr)
}

func getHostPortFromAddr(addr net.Addr) (host string, port int, err error) {
	host, portString, err := net.SplitHostPort(addr.String())
	if err != nil {
		return
	}
	port, err = strconv.Atoi(portString)
	return
}

// directForward is structure for RFC 4254 7.2 - can be used for "forwarded-tcpip" and "direct-tcpip"
type directForward struct {
	Host1 string
	Port1 uint32
	Host2 string
	Port2 uint32
}

func (p directForward) String() string {
	return fmt.Sprintf("%s:%d -> %s:%d", p.Host2, p.Port2, p.Host1, p.Port1)
}

// tcpipForward is structure for RFC 4254 7.1 "tcpip-forward" request
type tcpipForward struct {
	Host string
	Port uint32
}
