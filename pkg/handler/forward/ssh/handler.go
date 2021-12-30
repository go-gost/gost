package ssh

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/go-gost/gost/pkg/bypass"
	"github.com/go-gost/gost/pkg/chain"
	"github.com/go-gost/gost/pkg/handler"
	ssh_util "github.com/go-gost/gost/pkg/internal/util/ssh"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
	"golang.org/x/crypto/ssh"
)

// Applicable SSH Request types for Port Forwarding - RFC 4254 7.X
const (
	DirectForwardRequest       = "direct-tcpip"         // RFC 4254 7.2
	RemoteForwardRequest       = "tcpip-forward"        // RFC 4254 7.1
	ForwardedTCPReturnRequest  = "forwarded-tcpip"      // RFC 4254 7.2
	CancelRemoteForwardRequest = "cancel-tcpip-forward" // RFC 4254 7.1
)

func init() {
	registry.RegisterHandler("sshd", NewHandler)
}

type forwardHandler struct {
	bypass bypass.Bypass
	config *ssh.ServerConfig
	router *chain.Router
	logger logger.Logger
	md     metadata
}

func NewHandler(opts ...handler.Option) handler.Handler {
	options := &handler.Options{}
	for _, opt := range opts {
		opt(options)
	}

	return &forwardHandler{
		bypass: options.Bypass,
		router: options.Router,
		logger: options.Logger,
	}
}

func (h *forwardHandler) Init(md md.Metadata) (err error) {
	if err = h.parseMetadata(md); err != nil {
		return
	}

	config := &ssh.ServerConfig{
		PasswordCallback:  ssh_util.PasswordCallback(h.md.authenticator),
		PublicKeyCallback: ssh_util.PublicKeyCallback(h.md.authorizedKeys),
	}

	config.AddHostKey(h.md.signer)

	if h.md.authenticator == nil && len(h.md.authorizedKeys) == 0 {
		config.NoClientAuth = true
	}

	h.config = config

	return nil
}

func (h *forwardHandler) Handle(ctx context.Context, conn net.Conn) {
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

	sshConn, chans, reqs, err := ssh.NewServerConn(conn, h.config)
	if err != nil {
		h.logger.Error(err)
		return
	}

	h.handleForward(ctx, sshConn, chans, reqs)
}

func (h *forwardHandler) handleForward(ctx context.Context, conn ssh.Conn, chans <-chan ssh.NewChannel, reqs <-chan *ssh.Request) {
	quit := make(chan struct{})
	defer close(quit) // quit signal

	go func() {
		for req := range reqs {
			switch req.Type {
			case RemoteForwardRequest:
				go h.tcpipForwardRequest(conn, req, quit)
			default:
				h.logger.Warnf("unsupported request type: %s, want reply: %v", req.Type, req.WantReply)
				if req.WantReply {
					req.Reply(false, nil)
				}
			}
		}
	}()

	go func() {
		for newChannel := range chans {
			// Check the type of channel
			t := newChannel.ChannelType()
			switch t {
			case DirectForwardRequest:
				channel, requests, err := newChannel.Accept()
				if err != nil {
					h.logger.Warnf("could not accept channel: %s", err.Error())
					continue
				}
				p := directForward{}
				ssh.Unmarshal(newChannel.ExtraData(), &p)

				h.logger.Debug(p.String())

				if p.Host1 == "<nil>" {
					p.Host1 = ""
				}

				go ssh.DiscardRequests(requests)
				go h.directPortForwardChannel(ctx, channel, net.JoinHostPort(p.Host1, strconv.Itoa(int(p.Port1))))
			default:
				h.logger.Warnf("unsupported channel type: %s", t)
				newChannel.Reject(ssh.Prohibited, fmt.Sprintf("unsupported channel type: %s", t))
			}
		}
	}()

	conn.Wait()
}

func (h *forwardHandler) directPortForwardChannel(ctx context.Context, channel ssh.Channel, raddr string) {
	defer channel.Close()

	// log.Logf("[ssh-tcp] %s - %s", h.options.Node.Addr, raddr)

	/*
		if !Can("tcp", raddr, h.options.Whitelist, h.options.Blacklist) {
			log.Logf("[ssh-tcp] Unauthorized to tcp connect to %s", raddr)
			return
		}
	*/

	if h.bypass != nil && h.bypass.Contains(raddr) {
		h.logger.Infof("bypass %s", raddr)
		return
	}

	conn, err := h.router.Dial(ctx, "tcp", raddr)
	if err != nil {
		return
	}
	defer conn.Close()

	t := time.Now()
	h.logger.Infof("%s <-> %s", conn.LocalAddr(), conn.RemoteAddr())
	handler.Transport(conn, channel)
	h.logger.WithFields(map[string]interface{}{
		"duration": time.Since(t),
	}).Infof("%s >-< %s", conn.LocalAddr(), conn.RemoteAddr())
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

func getHostPortFromAddr(addr net.Addr) (host string, port int, err error) {
	host, portString, err := net.SplitHostPort(addr.String())
	if err != nil {
		return
	}
	port, err = strconv.Atoi(portString)
	return
}

// tcpipForward is structure for RFC 4254 7.1 "tcpip-forward" request
type tcpipForward struct {
	Host string
	Port uint32
}

func (h *forwardHandler) tcpipForwardRequest(sshConn ssh.Conn, req *ssh.Request, quit <-chan struct{}) {
	t := tcpipForward{}
	ssh.Unmarshal(req.Payload, &t)

	addr := net.JoinHostPort(t.Host, strconv.Itoa(int(t.Port)))

	/*
		if !Can("rtcp", addr, h.options.Whitelist, h.options.Blacklist) {
			log.Logf("[ssh-rtcp] Unauthorized to tcp bind to %s", addr)
			req.Reply(false, nil)
			return
		}
	*/

	// tie to the client connection
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		h.logger.Error(err)
		req.Reply(false, nil)
		return
	}
	defer ln.Close()

	h.logger.Debugf("bind on %s OK", ln.Addr())

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
		h.logger.Error(err)
		return
	}

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil { // Unable to accept new connection - listener is likely closed
				return
			}

			go func(conn net.Conn) {
				defer conn.Close()

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
					h.logger.Error("open forwarded channel: ", err)
					return
				}
				defer ch.Close()
				go ssh.DiscardRequests(reqs)

				t := time.Now()
				h.logger.Infof("%s <-> %s", conn.RemoteAddr(), conn.LocalAddr())
				handler.Transport(ch, conn)
				h.logger.WithFields(map[string]interface{}{
					"duration": time.Since(t),
				}).Infof("%s >-< %s", conn.RemoteAddr(), conn.LocalAddr())
			}(conn)
		}
	}()

	<-quit
}
