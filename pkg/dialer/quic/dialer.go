package quic

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/go-gost/gost/pkg/dialer"
	quic_util "github.com/go-gost/gost/pkg/internal/util/quic"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
	"github.com/lucas-clemente/quic-go"
)

func init() {
	registry.DialerRegistry().Register("quic", NewDialer)
}

type quicDialer struct {
	sessions     map[string]*quicSession
	sessionMutex sync.Mutex
	logger       logger.Logger
	md           metadata
	options      dialer.Options
}

func NewDialer(opts ...dialer.Option) dialer.Dialer {
	options := dialer.Options{}
	for _, opt := range opts {
		opt(&options)
	}

	return &quicDialer{
		sessions: make(map[string]*quicSession),
		logger:   options.Logger,
		options:  options,
	}
}

func (d *quicDialer) Init(md md.Metadata) (err error) {
	if err = d.parseMetadata(md); err != nil {
		return
	}

	return nil
}

// Multiplex implements dialer.Multiplexer interface.
func (d *quicDialer) Multiplex() bool {
	return true
}

func (d *quicDialer) Dial(ctx context.Context, addr string, opts ...dialer.DialOption) (conn net.Conn, err error) {
	var options dialer.DialOptions
	for _, opt := range opts {
		opt(&options)
	}

	d.sessionMutex.Lock()
	defer d.sessionMutex.Unlock()

	session, ok := d.sessions[addr]
	if !ok {
		var cc *net.UDPConn
		cc, err = net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
		if err != nil {
			return
		}
		conn = cc

		if d.md.cipherKey != nil {
			conn = quic_util.CipherConn(cc, d.md.cipherKey)
		}

		session = &quicSession{conn: conn}
		d.sessions[addr] = session
	}

	return session.conn, err
}

// Handshake implements dialer.Handshaker
func (d *quicDialer) Handshake(ctx context.Context, conn net.Conn, options ...dialer.HandshakeOption) (net.Conn, error) {
	opts := &dialer.HandshakeOptions{}
	for _, option := range options {
		option(opts)
	}

	d.sessionMutex.Lock()
	defer d.sessionMutex.Unlock()

	if d.md.handshakeTimeout > 0 {
		conn.SetDeadline(time.Now().Add(d.md.handshakeTimeout))
		defer conn.SetDeadline(time.Time{})
	}

	session, ok := d.sessions[opts.Addr]
	if session != nil && session.conn != conn {
		conn.Close()
		return nil, errors.New("quic: unrecognized connection")
	}
	if !ok || session.session == nil {
		s, err := d.initSession(ctx, opts.Addr, conn)
		if err != nil {
			d.logger.Error(err)
			conn.Close()
			delete(d.sessions, opts.Addr)
			return nil, err
		}
		session = s
		d.sessions[opts.Addr] = session
	}
	cc, err := session.GetConn()
	if err != nil {
		session.Close()
		delete(d.sessions, opts.Addr)
		return nil, err
	}

	return cc, nil
}

func (d *quicDialer) initSession(ctx context.Context, addr string, conn net.Conn) (*quicSession, error) {
	pc, ok := conn.(net.PacketConn)
	if !ok {
		return nil, errors.New("quic: wrong connection type")
	}

	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}
	quicConfig := &quic.Config{
		KeepAlive:            d.md.keepAlive,
		HandshakeIdleTimeout: d.md.handshakeTimeout,
		MaxIdleTimeout:       d.md.maxIdleTimeout,
		Versions: []quic.VersionNumber{
			quic.Version1,
			quic.VersionDraft29,
		},
	}

	tlsCfg := d.options.TLSConfig
	tlsCfg.NextProtos = []string{"http/3", "quic/v1"}

	session, err := quic.DialContext(ctx, pc, udpAddr, addr, tlsCfg, quicConfig)
	if err != nil {
		d.logger.Error(err)
		return nil, err
	}
	return &quicSession{conn: conn, session: session}, nil
}
