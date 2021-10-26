package tcp

import (
	"errors"
	"net"
	"strconv"
	"time"

	"github.com/go-gost/gost/pkg/components/internal/utils"
	"github.com/go-gost/gost/pkg/components/listener"
	"github.com/go-gost/gost/pkg/logger"
)

var (
	_ listener.Listener = (*Listener)(nil)
)

type Listener struct {
	md metadata
	net.Listener
	logger logger.Logger
}

func NewListener(opts ...listener.Option) *Listener {
	options := &listener.Options{}
	for _, opt := range opts {
		opt(options)
	}
	return &Listener{
		logger: options.Logger,
	}
}

func (l *Listener) Init(md listener.Metadata) (err error) {
	l.md, err = l.parseMetadata(md)
	if err != nil {
		return
	}

	laddr, err := net.ResolveTCPAddr("tcp", l.md.addr)
	if err != nil {
		return
	}
	ln, err := net.ListenTCP("tcp", laddr)
	if err != nil {
		return
	}

	if l.md.keepAlive {
		l.Listener = &utils.TCPKeepAliveListener{
			TCPListener:     ln,
			KeepAlivePeriod: l.md.keepAlivePeriod,
		}
		return
	}

	l.Listener = ln
	return
}

func (l *Listener) parseMetadata(md listener.Metadata) (m metadata, err error) {
	if val, ok := md[addr]; ok {
		m.addr = val
	} else {
		err = errors.New("missing address")
		return
	}

	m.keepAlive = true
	if val, ok := md[keepAlive]; ok {
		m.keepAlive, _ = strconv.ParseBool(val)
	}

	if val, ok := md[keepAlivePeriod]; ok {
		m.keepAlivePeriod, _ = time.ParseDuration(val)
	}

	return
}
