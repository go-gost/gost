package v5

import (
	"crypto/tls"
	"net"
	"net/url"

	"github.com/go-gost/gosocks5"
	"github.com/go-gost/gost/pkg/common/util/socks"
	"github.com/go-gost/gost/pkg/logger"
)

type clientSelector struct {
	methods   []uint8
	User      *url.Userinfo
	TLSConfig *tls.Config
	logger    logger.Logger
}

func (s *clientSelector) Methods() []uint8 {
	s.logger.Debug("methods: ", s.methods)
	return s.methods
}

func (s *clientSelector) AddMethod(methods ...uint8) {
	s.methods = append(s.methods, methods...)
}

func (s *clientSelector) Select(methods ...uint8) (method uint8) {
	return
}

func (s *clientSelector) OnSelected(method uint8, conn net.Conn) (net.Conn, error) {
	s.logger.Debug("method selected: ", method)

	switch method {
	case socks.MethodTLS:
		conn = tls.Client(conn, s.TLSConfig)

	case gosocks5.MethodUserPass, socks.MethodTLSAuth:
		if method == socks.MethodTLSAuth {
			conn = tls.Client(conn, s.TLSConfig)
		}

		var username, password string
		if s.User != nil {
			username = s.User.Username()
			password, _ = s.User.Password()
		}

		req := gosocks5.NewUserPassRequest(gosocks5.UserPassVer, username, password)
		if err := req.Write(conn); err != nil {
			s.logger.Error(err)
			return nil, err
		}
		s.logger.Debug(req)

		resp, err := gosocks5.ReadUserPassResponse(conn)
		if err != nil {
			s.logger.Error(err)
			return nil, err
		}
		s.logger.Debug(resp)

		if resp.Status != gosocks5.Succeeded {
			return nil, gosocks5.ErrAuthFailure
		}
	case gosocks5.MethodNoAcceptable:
		return nil, gosocks5.ErrBadMethod
	}

	return conn, nil
}
