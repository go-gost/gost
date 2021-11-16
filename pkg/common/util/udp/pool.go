package udp

import (
	"sync"
	"time"

	"github.com/go-gost/gost/pkg/logger"
)

type ConnPool struct {
	m      sync.Map
	ttl    time.Duration
	closed chan struct{}
	logger logger.Logger
}

func NewConnPool(ttl time.Duration) *ConnPool {
	p := &ConnPool{
		ttl:    ttl,
		closed: make(chan struct{}),
	}
	go p.idleCheck()
	return p
}

func (p *ConnPool) WithLogger(logger logger.Logger) *ConnPool {
	p.logger = logger
	return p
}

func (p *ConnPool) Get(key interface{}) (c *Conn, ok bool) {
	v, ok := p.m.Load(key)
	if ok {
		c, ok = v.(*Conn)
	}
	return
}

func (p *ConnPool) Set(key interface{}, c *Conn) {
	p.m.Store(key, c)
}

func (p *ConnPool) Delete(key interface{}) {
	p.m.Delete(key)
}

func (p *ConnPool) Close() {
	select {
	case <-p.closed:
		return
	default:
	}

	close(p.closed)

	p.m.Range(func(k, v interface{}) bool {
		if c, ok := v.(*Conn); ok && c != nil {
			c.Close()
		}
		return true
	})
}

func (p *ConnPool) idleCheck() {
	ticker := time.NewTicker(p.ttl)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			size := 0
			idles := 0
			p.m.Range(func(key, value interface{}) bool {
				c, ok := value.(*Conn)
				if !ok || c == nil {
					p.Delete(key)
					return true
				}
				size++

				if c.IsIdle() {
					idles++
					p.Delete(key)
					c.Close()
					return true
				}

				c.SetIdle(true)

				return true
			})

			if idles > 0 {
				p.logger.Debugf("connection pool: size=%d, idle=%d", size, idles)
			}
		case <-p.closed:
			return
		}
	}
}
