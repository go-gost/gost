package tcp

import (
	"net"
	"strconv"
	"time"

	"github.com/go-gost/gost/pkg/components/internal/utils"
	"github.com/go-gost/gost/pkg/components/listener"
	"github.com/go-gost/gost/pkg/logger"
	"github.com/go-gost/gost/pkg/registry"
)

func init() {
	registry.RegisterListener("tcp", NewListener)
}

type Listener struct {
	addr string
	md   metadata
	net.Listener
	logger logger.Logger
}

func NewListener(opts ...listener.Option) listener.Listener {
	options := &listener.Options{}
	for _, opt := range opts {
		opt(options)
	}
	return &Listener{
		addr:   options.Addr,
		logger: options.Logger,
	}
}

func (l *Listener) Init(md listener.Metadata) (err error) {
	l.md, err = l.parseMetadata(md)
	if err != nil {
		return
	}

	laddr, err := net.ResolveTCPAddr("tcp", l.addr)
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
	m.keepAlive = true
	if val, ok := md[keepAlive]; ok {
		m.keepAlive, _ = strconv.ParseBool(val)
	}

	if val, ok := md[keepAlivePeriod]; ok {
		m.keepAlivePeriod, _ = time.ParseDuration(val)
	}

	return
}
