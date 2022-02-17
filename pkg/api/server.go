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

	router := r.Group("")
	if options.pathPrefix != "" {
		router = router.Group(options.pathPrefix)
	}

	router.StaticFS("/docs", http.FS(swaggerDoc))

	config := router.Group("/config")
	config.Use(mwBasicAuth(options.auther))
	registerConfig(config)

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

func registerConfig(config *gin.RouterGroup) {
	config.GET("", getConfig)
	config.POST("", saveConfig)

	config.POST("/services", createService)
	config.PUT("/services/:service", updateService)
	config.DELETE("/services/:service", deleteService)

	config.POST("/chains", createChain)
	config.PUT("/chains/:chain", updateChain)
	config.DELETE("/chains/:chain", deleteChain)

	config.POST("/authers", createAuther)
	config.PUT("/authers/:auther", updateAuther)
	config.DELETE("/authers/:auther", deleteAuther)

	config.POST("/admissions", createAdmission)
	config.PUT("/admissions/:admission", updateAdmission)
	config.DELETE("/admissions/:admission", deleteAdmission)

	config.POST("/bypasses", createBypass)
	config.PUT("/bypasses/:bypass", updateBypass)
	config.DELETE("/bypasses/:bypass", deleteBypass)

	config.POST("/resolvers", createResolver)
	config.PUT("/resolvers/:resolver", updateResolver)
	config.DELETE("/resolvers/:resolver", deleteResolver)

	config.POST("/hosts", createHosts)
	config.PUT("/hosts/:hosts", updateHosts)
	config.DELETE("/hosts/:hosts", deleteHosts)
}
