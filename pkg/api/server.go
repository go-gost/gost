package api

import (
	"net"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go-gost/gost/pkg/auth"
)

type options struct {
	accessLog  bool
	pathPrefix string
	auther     auth.Authenticator
}

type Option func(*options)

func PathPrefixOption(pathPrefix string) Option {
	return func(o *options) {
		o.pathPrefix = pathPrefix
	}
}

func AccessLogOption(enable bool) Option {
	return func(o *options) {
		o.accessLog = enable
	}
}

func AutherOption(auther auth.Authenticator) Option {
	return func(o *options) {
		o.auther = auther
	}
}

type Server struct {
	s  *http.Server
	ln net.Listener
}

func NewServer(addr string, opts ...Option) (*Server, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	var options options
	for _, opt := range opts {
		opt(&options)
	}

	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(
		cors.New((cors.Config{
			AllowAllOrigins: true,
			AllowMethods:    []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowHeaders:    []string{"*"},
		})),
		gin.Recovery(),
	)
	if options.accessLog {
		r.Use(mwLogger())
	}
	if options.auther != nil {
		r.Use(mwBasicAuth(options.auther))
	}

	router := r.Group("")
	if options.pathPrefix != "" {
		router = router.Group(options.pathPrefix)
	}
	register(router)

	return &Server{
		s: &http.Server{
			Handler: r,
		},
		ln: ln,
	}, nil
}

func (s *Server) Serve() error {
	return s.s.Serve(s.ln)
}

func (s *Server) Addr() net.Addr {
	return s.ln.Addr()
}

func (s *Server) Close() error {
	return s.s.Close()
}
