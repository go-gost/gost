package tcp

import (
	"errors"
	"net"
	"strconv"
	"time"

	"github.com/go-gost/gost/server/listener"
)

type Listener struct {
	md metadata
	net.Listener
}

func NewTCPListener() *Listener {
	return &Listener{}
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
		l.Listener = &keepAliveListener{
			TCPListener:     ln,
			keepAlivePeriod: l.md.keepAlivePeriod,
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
		err = errors.New("tcp listener: missing address")
		return
	}

	m.keepAlive = true
	if val, ok := md[keepAlive]; ok {
		m.keepAlive, _ = strconv.ParseBool(val)
	}

	if val, ok := md[keepAlivePeriod]; ok {
		m.keepAlivePeriod, _ = time.ParseDuration(val)
	}
	if m.keepAlivePeriod <= 0 {
		m.keepAlivePeriod = defaultKeepAlivePeriod
	}

	return
}

type keepAliveListener struct {
	keepAlivePeriod time.Duration
	*net.TCPListener
}

func (l *keepAliveListener) Accept() (c net.Conn, err error) {
	tc, err := l.AcceptTCP()
	if err != nil {
		return
	}

	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(l.keepAlivePeriod)

	return tc, nil
}
