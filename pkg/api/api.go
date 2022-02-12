package api

import (
	"embed"
	"net/http"

	"github.com/gin-gonic/gin"
)

var (
	//go:embed swagger.yaml
	swaggerDoc embed.FS
)

func register(r *gin.RouterGroup) {
	r.StaticFS("/docs", http.FS(swaggerDoc))

	config := r.Group("/config")
	{
		config.GET("", getConfig)

		config.POST("/services", createService)
		config.PUT("/services/:service", updateService)
		config.DELETE("/services/:service", deleteService)

		config.POST("/chains", createChain)
		config.PUT("/chains/:chain", updateChain)
		config.DELETE("/chains/:chain", deleteChain)

		config.POST("/authers", createAuther)
		config.PUT("/authers/:auther", updateAuther)
		config.DELETE("/authers/:auther", deleteAuther)

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
}

type Response struct {
	Code int    `json:"code,omitempty"`
	Msg  string `json:"msg,omitempty"`
}
