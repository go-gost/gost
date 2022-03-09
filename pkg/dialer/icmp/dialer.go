package quic

import (
	"context"
	"math"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/go-gost/gost/pkg/dialer"
	icmp_pkg "github.com/go-gost/gost/pkg/internal/util/icmp"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
	"github.com/lucas-clemente/quic-go"
	"golang.org/x/net/icmp"
)

func init() {
	registry.DialerRegistry().Register("icmp", NewDialer)
}

type icmpDialer struct {
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

	return &icmpDialer{
		sessions: make(map[string]*quicSession),
		logger:   options.Logger,
		options:  options,
	}
}

func (d *icmpDialer) Init(md md.Metadata) (err error) {
	if err = d.parseMetadata(md); err != nil {
		return
	}

	return nil
}

func (d *icmpDialer) Dial(ctx context.Context, addr string, opts ...dialer.DialOption) (conn net.Conn, err error) {
	if _, _, err := net.SplitHostPort(addr); err != nil {
		addr = net.JoinHostPort(addr, "0")
	}

	raddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}

	d.sessionMutex.Lock()
	defer d.sessionMutex.Unlock()

	session, ok := d.sessions[addr]
	if !ok {
		options := &dialer.DialOptions{}
		for _, opt := range opts {
			opt(options)
		}

		var pc net.PacketConn
		pc, err = icmp.ListenPacket("ip4:icmp", "")
		if err != nil {
			return
		}

		id := raddr.Port
		if id == 0 {
			id = rand.New(rand.NewSource(time.Now().UnixNano())).Intn(math.MaxUint16) + 1
			raddr.Port = id
		}
		pc = icmp_pkg.ClientConn(pc, id)

		session, err = d.initSession(ctx, raddr, pc)
		if err != nil {
			d.logger.Error(err)
			pc.Close()
			return nil, err
		}

		d.sessions[addr] = session
	}

	conn, err = session.GetConn()
	if err != nil {
		session.Close()
		delete(d.sessions, addr)
		return nil, err
	}

	return
}

func (d *icmpDialer) initSession(ctx context.Context, addr net.Addr, conn net.PacketConn) (*quicSession, error) {
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

	session, err := quic.DialContext(ctx, conn, addr, addr.String(), tlsCfg, quicConfig)
	if err != nil {
		return nil, err
	}
	return &quicSession{session: session}, nil
}

// Multiplex implements dialer.Multiplexer interface.
func (d *icmpDialer) Multiplex() bool {
	return true
}
