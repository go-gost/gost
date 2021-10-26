package service

import (
	"context"
	"net"
	"time"

	"github.com/go-gost/gost/pkg/components/handler"
	"github.com/go-gost/gost/pkg/components/listener"
)

type Service struct {
	listener listener.Listener
	handler  handler.Handler
}

func (s *Service) WithListener(ln listener.Listener) *Service {
	s.listener = ln
	return s
}

func (s *Service) WithHandler(h handler.Handler) *Service {
	s.handler = h
	return s
}

func (s *Service) Addr() net.Addr {
	return s.listener.Addr()
}

func (s *Service) Run() error {
	return s.serve()
}

func (s *Service) Close() error {
	return s.listener.Close()
}

func (s *Service) serve() error {
	var tempDelay time.Duration
	for {
		conn, e := s.listener.Accept()
		if e != nil {
			if ne, ok := e.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				// log.Logf("server: Accept error: %v; retrying in %v", e, tempDelay)
				time.Sleep(tempDelay)
				continue
			}
			return e
		}
		tempDelay = 0

		go s.handler.Handle(context.Background(), conn)
	}
}
