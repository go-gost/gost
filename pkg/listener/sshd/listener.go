package ssh

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/go-gost/gost/pkg/common/admission"
	"github.com/go-gost/gost/pkg/common/metrics"
	ssh_util "github.com/go-gost/gost/pkg/internal/util/ssh"
	sshd_util "github.com/go-gost/gost/pkg/internal/util/sshd"
	"github.com/go-gost/gost/pkg/listener"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
	"golang.org/x/crypto/ssh"
)

// Applicable SSH Request types for Port Forwarding - RFC 4254 7.X
const (
	DirectForwardRequest = "direct-tcpip"  // RFC 4254 7.2
	RemoteForwardRequest = "tcpip-forward" // RFC 4254 7.1
)

func init() {
	registry.ListenerRegistry().Register("sshd", NewListener)
}

type sshdListener struct {
	net.Listener
	config  *ssh.ServerConfig
	cqueue  chan net.Conn
	errChan chan error
	logger  logger.Logger
	md      metadata
	options listener.Options
}

func NewListener(opts ...listener.Option) listener.Listener {
	options := listener.Options{}
	for _, opt := range opts {
		opt(&options)
	}
	return &sshdListener{
		logger:  options.Logger,
		options: options,
	}
}

func (l *sshdListener) Init(md md.Metadata) (err error) {
	if err = l.parseMetadata(md); err != nil {
		return
	}

	ln, err := net.Listen("tcp", l.options.Addr)
	if err != nil {
		return err
	}

	ln = metrics.WrapListener(l.options.Service, ln)
	ln = admission.WrapListener(l.options.Admission, ln)
	l.Listener = ln

	config := &ssh.ServerConfig{
		PasswordCallback:  ssh_util.PasswordCallback(l.options.Auther),
		PublicKeyCallback: ssh_util.PublicKeyCallback(l.md.authorizedKeys),
	}
	config.AddHostKey(l.md.signer)
	if l.options.Auther == nil && len(l.md.authorizedKeys) == 0 {
		config.NoClientAuth = true
	}

	l.config = config
	l.cqueue = make(chan net.Conn, l.md.backlog)
	l.errChan = make(chan error, 1)

	go l.listenLoop()

	return
}

func (l *sshdListener) Accept() (conn net.Conn, err error) {
	var ok bool
	select {
	case conn = <-l.cqueue:
	case err, ok = <-l.errChan:
		if !ok {
			err = listener.ErrClosed
		}
	}
	return
}

func (l *sshdListener) listenLoop() {
	for {
		conn, err := l.Listener.Accept()
		if err != nil {
			l.logger.Error("accept:", err)
			l.errChan <- err
			close(l.errChan)
			return
		}
		go l.serveConn(conn)
	}
}

func (l *sshdListener) serveConn(conn net.Conn) {
	start := time.Now()
	l.logger.Infof("%s <> %s", conn.RemoteAddr(), conn.LocalAddr())
	defer func() {
		l.logger.WithFields(map[string]any{
			"duration": time.Since(start),
		}).Infof("%s >< %s", conn.RemoteAddr(), conn.LocalAddr())
	}()

	sc, chans, reqs, err := ssh.NewServerConn(conn, l.config)
	if err != nil {
		l.logger.Error(err)
		conn.Close()
		return
	}
	defer sc.Close()

	go func() {
		for newChannel := range chans {
			// Check the type of channel
			t := newChannel.ChannelType()
			switch t {
			case DirectForwardRequest:
				channel, requests, err := newChannel.Accept()
				if err != nil {
					l.logger.Warnf("could not accept channel: %s", err.Error())
					continue
				}
				p := directForward{}
				ssh.Unmarshal(newChannel.ExtraData(), &p)

				l.logger.Debug(p.String())

				if p.Host1 == "<nil>" {
					p.Host1 = ""
				}

				go ssh.DiscardRequests(requests)
				cc := sshd_util.NewDirectForwardConn(sc, channel, net.JoinHostPort(p.Host1, strconv.Itoa(int(p.Port1))))

				select {
				case l.cqueue <- cc:
				default:
					l.logger.Warnf("connection queue is full, client %s discarded", conn.RemoteAddr())
					newChannel.Reject(ssh.ResourceShortage, "connection queue is full")
					cc.Close()
				}

			default:
				l.logger.Warnf("unsupported channel type: %s", t)
				newChannel.Reject(ssh.UnknownChannelType, fmt.Sprintf("unsupported channel type: %s", t))
			}
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		for req := range reqs {
			switch req.Type {
			case RemoteForwardRequest:
				cc := sshd_util.NewRemoteForwardConn(ctx, sc, req)

				select {
				case l.cqueue <- cc:
				default:
					l.logger.Warnf("connection queue is full, client %s discarded", conn.RemoteAddr())
					req.Reply(false, []byte("connection queue is full"))
					cc.Close()
				}
			default:
				l.logger.Warnf("unsupported request type: %s, want reply: %v", req.Type, req.WantReply)
				req.Reply(false, nil)
			}
		}
	}()
	sc.Wait()
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
