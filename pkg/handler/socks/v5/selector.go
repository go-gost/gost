package v5

import (
	"crypto/tls"
	"net"

	"github.com/go-gost/gosocks5"
	"github.com/go-gost/gost/pkg/auth"
	"github.com/go-gost/gost/pkg/internal/utils/socks"
	"github.com/go-gost/gost/pkg/logger"
)

type serverSelector struct {
	methods       []uint8
	Authenticator auth.Authenticator
	TLSConfig     *tls.Config
	logger        logger.Logger
	noTLS         bool
}

func (selector *serverSelector) Methods() []uint8 {
	return selector.methods
}

func (s *serverSelector) Select(methods ...uint8) (method uint8) {
	if s.logger.IsLevelEnabled(logger.DebugLevel) {
		s.logger.Debugf("%d %d %v", gosocks5.Ver5, len(methods), methods)
	}
	method = gosocks5.MethodNoAuth
	for _, m := range methods {
		if m == socks.MethodTLS && !s.noTLS {
			method = m
			break
		}
	}

	// when Authenticator is set, auth is mandatory
	if s.Authenticator != nil {
		if method == gosocks5.MethodNoAuth {
			method = gosocks5.MethodUserPass
		}
		if method == socks.MethodTLS && !s.noTLS {
			method = socks.MethodTLSAuth
		}
	}

	return
}

func (s *serverSelector) OnSelected(method uint8, conn net.Conn) (net.Conn, error) {
	if s.logger.IsLevelEnabled(logger.DebugLevel) {
		s.logger.Debugf("%d %d", gosocks5.Ver5, method)
	}
	switch method {
	case socks.MethodTLS:
		conn = tls.Server(conn, s.TLSConfig)

	case gosocks5.MethodUserPass, socks.MethodTLSAuth:
		if method == socks.MethodTLSAuth {
			conn = tls.Server(conn, s.TLSConfig)
		}

		req, err := gosocks5.ReadUserPassRequest(conn)
		if err != nil {
			s.logger.Error(err)
			return nil, err
		}
		if s.logger.IsLevelEnabled(logger.DebugLevel) {
			s.logger.Debug(req.String())
		}

		if s.Authenticator != nil &&
			!s.Authenticator.Authenticate(req.Username, req.Password) {
			resp := gosocks5.NewUserPassResponse(gosocks5.UserPassVer, gosocks5.Failure)
			if err := resp.Write(conn); err != nil {
				s.logger.Error(err)
				return nil, err
			}
			if s.logger.IsLevelEnabled(logger.DebugLevel) {
				s.logger.Info(resp.String())
			}
			return nil, gosocks5.ErrAuthFailure
		}

		resp := gosocks5.NewUserPassResponse(gosocks5.UserPassVer, gosocks5.Succeeded)
		if err := resp.Write(conn); err != nil {
			s.logger.Error(err)
			return nil, err
		}
		if s.logger.IsLevelEnabled(logger.DebugLevel) {
			s.logger.Debug(resp.String())
		}
	case gosocks5.MethodNoAcceptable:
		return nil, gosocks5.ErrBadMethod
	}

	return conn, nil
}
