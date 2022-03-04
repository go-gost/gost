package ftcp

import (
	"net"
	"sync"
	"sync/atomic"

	"github.com/go-gost/gost/pkg/common/metrics"
	"github.com/go-gost/gost/pkg/listener"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
	"github.com/xtaci/tcpraw"
)

func init() {
	registry.ListenerRegistry().Register("ftcp", NewListener)
}

type ftcpListener struct {
	conn     net.PacketConn
	connChan chan net.Conn
	errChan  chan error
	connPool connPool
	logger   logger.Logger
	md       metadata
	options  listener.Options
}

func NewListener(opts ...listener.Option) listener.Listener {
	options := listener.Options{}
	for _, opt := range opts {
		opt(&options)
	}
	return &ftcpListener{
		logger:  options.Logger,
		options: options,
	}
}

func (l *ftcpListener) Init(md md.Metadata) (err error) {
	if err = l.parseMetadata(md); err != nil {
		return
	}

	l.conn, err = tcpraw.Listen("tcp", l.options.Addr)
	if err != nil {
		return
	}

	l.connChan = make(chan net.Conn, l.md.connQueueSize)
	l.errChan = make(chan error, 1)

	go l.listenLoop()

	return
}

func (l *ftcpListener) Accept() (conn net.Conn, err error) {
	var ok bool
	select {
	case conn = <-l.connChan:
		conn = metrics.WrapConn(l.options.Service, conn)
	case err, ok = <-l.errChan:
		if !ok {
			err = listener.ErrClosed
		}
	}
	return
}

func (l *ftcpListener) Close() error {
	err := l.conn.Close()
	l.connPool.Range(func(k any, v *serverConn) bool {
		v.Close()
		return true
	})
	return err
}

func (l *ftcpListener) Addr() net.Addr {
	return l.conn.LocalAddr()
}

func (l *ftcpListener) listenLoop() {
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

func (l *ftcpListener) parseMetadata(md md.Metadata) (err error) {
	return
}

type connPool struct {
	size int64
	m    sync.Map
}

func (p *connPool) Get(key any) (conn *serverConn, ok bool) {
	v, ok := p.m.Load(key)
	if ok {
		conn, ok = v.(*serverConn)
	}
	return
}

func (p *connPool) Set(key any, conn *serverConn) {
	p.m.Store(key, conn)
	atomic.AddInt64(&p.size, 1)
}

func (p *connPool) Delete(key any) {
	p.m.Delete(key)
	atomic.AddInt64(&p.size, -1)
}

func (p *connPool) Range(f func(key any, value *serverConn) bool) {
	p.m.Range(func(k, v any) bool {
		return f(k, v.(*serverConn))
	})
}

func (p *connPool) Size() int64 {
	return atomic.LoadInt64(&p.size)
}
