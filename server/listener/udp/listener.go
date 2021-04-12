package udp

import (
	"errors"
	"net"
	"sync"
	"sync/atomic"

	"github.com/go-gost/gost/logger"
	"github.com/go-gost/gost/server/listener"
)

var (
	_ listener.Listener = (*Listener)(nil)
)

type Listener struct {
	md       metadata
	conn     net.PacketConn
	connChan chan net.Conn
	errChan  chan error
	connPool connPool
	logger   logger.Logger
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

	laddr, err := net.ResolveUDPAddr("udp", l.md.addr)
	if err != nil {
		return
	}

	var conn net.PacketConn
	conn, err = net.ListenUDP("udp", laddr)
	if err != nil {
		return
	}

	l.conn = conn
	l.connChan = make(chan net.Conn, l.md.connQueueSize)
	l.errChan = make(chan error, 1)

	go l.listenLoop()

	return
}

func (l *Listener) Accept() (conn net.Conn, err error) {
	var ok bool
	select {
	case conn = <-l.connChan:
	case err, ok = <-l.errChan:
		if !ok {
			err = listener.ErrClosed
		}
	}
	return
}

func (l *Listener) Close() error {
	return l.conn.Close()
}

func (l *Listener) Addr() net.Addr {
	return l.conn.LocalAddr()
}

func (l *Listener) listenLoop() {
	for {
		b := make([]byte, l.md.readBufferSize)

		n, raddr, err := l.conn.ReadFrom(b)
		if err != nil {
			l.logger.Error("accept:", err)
			l.errChan <- err
			close(l.errChan)
			return
		}

		conn, ok := l.connPool.Get(raddr.String())
		if !ok {
			conn = newServerConn(l.conn, raddr,
				&serverConnConfig{
					ttl:   l.md.ttl,
					qsize: l.md.readQueueSize,
					onClose: func() {
						l.connPool.Delete(raddr.String())
					},
				})

			select {
			case l.connChan <- conn:
				l.connPool.Set(raddr.String(), conn)
			default:
				conn.Close()
				l.logger.Error("connection queue is full")
			}
		}

		if err := conn.send(b[:n]); err != nil {
			l.logger.Warn("data discarded:", err)
		}
		l.logger.Debug("recv", n)
	}
}

func (l *Listener) parseMetadata(md listener.Metadata) (m metadata, err error) {
	if val, ok := md[addr]; ok {
		m.addr = val
	} else {
		err = errors.New("missing address")
		return
	}

	return
}

type connPool struct {
	size int64
	m    sync.Map
}

func (p *connPool) Get(key interface{}) (conn *serverConn, ok bool) {
	v, ok := p.m.Load(key)
	if ok {
		conn, ok = v.(*serverConn)
	}
	return
}

func (p *connPool) Set(key interface{}, conn *serverConn) {
	p.m.Store(key, conn)
	atomic.AddInt64(&p.size, 1)
}

func (p *connPool) Delete(key interface{}) {
	p.m.Delete(key)
	atomic.AddInt64(&p.size, -1)
}

func (p *connPool) Range(f func(key interface{}, value *serverConn) bool) {
	p.m.Range(func(k, v interface{}) bool {
		return f(k, v.(*serverConn))
	})
}

func (p *connPool) Size() int64 {
	return atomic.LoadInt64(&p.size)
}
