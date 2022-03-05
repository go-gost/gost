package service

import (
	"context"
	"net"
	"time"

	"github.com/go-gost/gost/pkg/admission"
	"github.com/go-gost/gost/pkg/handler"
	"github.com/go-gost/gost/pkg/listener"
	"github.com/go-gost/gost/pkg/logger"
	"github.com/go-gost/gost/pkg/metrics"
)

type options struct {
	admission admission.Admission
	logger    logger.Logger
}

type Option func(opts *options)

func AdmissionOption(admission admission.Admission) Option {
	return func(opts *options) {
		opts.admission = admission
	}
}

func LoggerOption(logger logger.Logger) Option {
	return func(opts *options) {
		opts.logger = logger
	}
}

type Service interface {
	Serve() error
	Addr() net.Addr
	Close() error
}

type service struct {
	name     string
	listener listener.Listener
	handler  handler.Handler
	options  options
}

func NewService(name string, ln listener.Listener, h handler.Handler, opts ...Option) Service {
	var options options
	for _, opt := range opts {
		opt(&options)
	}
	return &service{
		name:     name,
		listener: ln,
		handler:  h,
		options:  options,
	}
}

func (s *service) Addr() net.Addr {
	return s.listener.Addr()
}

func (s *service) Close() error {
	return s.listener.Close()
}

func (s *service) Serve() error {
	metrics.Services().Inc()
	defer metrics.Services().Dec()

	var tempDelay time.Duration
	for {
		conn, e := s.listener.Accept()
		if e != nil {
			if ne, ok := e.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 1 * time.Second
				} else {
					tempDelay *= 2
				}
				if max := 5 * time.Second; tempDelay > max {
					tempDelay = max
				}
				s.options.logger.Warnf("accept: %v, retrying in %v", e, tempDelay)
				time.Sleep(tempDelay)
				continue
			}
			s.options.logger.Errorf("accept: %v", e)
			return e
		}
		tempDelay = 0

		if s.options.admission != nil &&
			!s.options.admission.Admit(conn.RemoteAddr().String()) {
			s.options.logger.Infof("admission: %s is denied", conn.RemoteAddr())
			conn.Close()
			continue
		}

		go func() {
			metrics.Requests(s.name).Inc()

			metrics.RequestsInFlight(s.name).Inc()
			defer metrics.RequestsInFlight(s.name).Dec()

			start := time.Now()
			defer func() {
				metrics.RequestSeconds(s.name).Observe(time.Since(start).Seconds())
			}()

			if err := s.handler.Handle(context.Background(), conn); err != nil {
				metrics.HandlerErrors(s.name).Inc()
			}
		}()
	}
}
