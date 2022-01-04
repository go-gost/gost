package ssh

import (
	"fmt"
	"net"

	auth_util "github.com/go-gost/gost/pkg/common/util/auth"
	ssh_util "github.com/go-gost/gost/pkg/internal/util/ssh"
	"github.com/go-gost/gost/pkg/listener"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
	"golang.org/x/crypto/ssh"
)

func init() {
	registry.RegisterListener("ssh", NewListener)
}

type sshListener struct {
	addr string
	net.Listener
	config  *ssh.ServerConfig
	cqueue  chan net.Conn
	errChan chan error
	logger  logger.Logger
	md      metadata
	options listener.Options
}

func NewListener(opts ...listener.Option) listener.Listener {
	options := &listener.Options{}
	for _, opt := range opts {
		opt(options)
	}
	return &sshListener{
		addr:   options.Addr,
		logger: options.Logger,
	}
}

func (l *sshListener) Init(md md.Metadata) (err error) {
	if err = l.parseMetadata(md); err != nil {
		return
	}

	ln, err := net.Listen("tcp", l.addr)
	if err != nil {
		return err
	}

	l.Listener = ln

	authenticator := auth_util.AuthFromUsers(l.options.Auths...)
	config := &ssh.ServerConfig{
		PasswordCallback:  ssh_util.PasswordCallback(authenticator),
		PublicKeyCallback: ssh_util.PublicKeyCallback(l.md.authorizedKeys),
	}
	config.AddHostKey(l.md.signer)
	if authenticator == nil && len(l.md.authorizedKeys) == 0 {
		config.NoClientAuth = true
	}

	l.config = config
	l.cqueue = make(chan net.Conn, l.md.backlog)
	l.errChan = make(chan error, 1)

	go l.listenLoop()

	return
}

func (l *sshListener) Accept() (conn net.Conn, err error) {
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

func (l *sshListener) listenLoop() {
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

func (l *sshListener) serveConn(conn net.Conn) {
	sc, chans, reqs, err := ssh.NewServerConn(conn, l.config)
	if err != nil {
		l.logger.Error(err)
		conn.Close()
		return
	}
	defer sc.Close()

	go ssh.DiscardRequests(reqs)
	go func() {
		for newChannel := range chans {
			// Check the type of channel
			t := newChannel.ChannelType()
			switch t {
			case ssh_util.GostSSHTunnelRequest:
				channel, requests, err := newChannel.Accept()
				if err != nil {
					l.logger.Warnf("could not accept channel: %s", err.Error())
					continue
				}

				go ssh.DiscardRequests(requests)
				cc := ssh_util.NewConn(conn, channel)
				select {
				case l.cqueue <- cc:
				default:
					cc.Close()
					l.logger.Warnf("connection queue is full, client %s discarded", conn.RemoteAddr())
				}

			default:
				l.logger.Warnf("unsupported channel type: %s", t)
				newChannel.Reject(ssh.UnknownChannelType, fmt.Sprintf("unsupported channel type: %s", t))
			}
		}
	}()

	sc.Wait()
}
