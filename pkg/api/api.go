package api

import (
	"embed"
	"net"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go-gost/gost/pkg/config"
	"github.com/go-gost/gost/pkg/logger"
)

var (
	apiServer = &http.Server{}

	//go:embed swagger.yaml
	swaggerDoc embed.FS
)

func Init(cfg *config.APIConfig) {
	gin.SetMode(gin.ReleaseMode)

	if cfg == nil {
		cfg = &config.APIConfig{}
	}

	r := gin.New()
	r.Use(
		cors.New((cors.Config{
			AllowAllOrigins: true,
			AllowMethods:    []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowHeaders:    []string{"*"},
		})),
		gin.Recovery(),
	)
	if cfg.AccessLog {
		r.Use(loggerHandler)
	}

	router := r.Group("")
	if cfg.PathPrefix != "" {
		router = router.Group(cfg.PathPrefix)
	}

	router.StaticFS("/docs", http.FS(swaggerDoc))

	config := router.Group("/config")
	{
		config.GET("", getConfig)

		config.POST("/services", createService)
		config.PUT("/services/:service", updateService)
		config.DELETE("/services/:service", deleteService)

		config.POST("/chains", createChain)
		config.PUT("/chains/:chain", updateChain)
		config.DELETE("/chains/:chain", deleteChain)

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

	apiServer.Handler = r
}

func Run(ln net.Listener) error {
	return apiServer.Serve(ln)
}

func loggerHandler(ctx *gin.Context) {
	// start time
	startTime := time.Now()
	// Processing request
	ctx.Next()
	duration := time.Since(startTime)

	logger.Default().WithFields(map[string]interface{}{
		"kind":     "api",
		"method":   ctx.Request.Method,
		"uri":      ctx.Request.RequestURI,
		"code":     ctx.Writer.Status(),
		"client":   ctx.ClientIP(),
		"duration": duration,
	}).Infof("| %3d | %13v | %15s | %-7s %s",
		ctx.Writer.Status(), duration, ctx.ClientIP(), ctx.Request.Method, ctx.Request.RequestURI)
}

type Response struct {
	Code int    `json:"code,omitempty"`
	Msg  string `json:"msg,omitempty"`
}
