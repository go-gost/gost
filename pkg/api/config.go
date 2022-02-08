package api

import (
	"bytes"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-gost/gost/pkg/config"
)

// swagger:parameters getConfigRequest
type getConfigRequest struct {
	// output format, one of yaml|json, default is json.
	// in: query
	Format string `form:"format" json:"format"`
}

// successful operation.
// swagger:response getConfigResponse
type getConfigResponse struct {
	Config *config.Config
}

func getConfig(ctx *gin.Context) {
	// swagger:route GET /config ConfigManagement getConfigRequest
	//
	// Get current config.
	//
	//     Responses:
	//       200: getConfigResponse

	var req getConfigRequest
	ctx.ShouldBindQuery(&req)

	var resp getConfigResponse
	resp.Config = config.Global()

	buf := &bytes.Buffer{}
	switch req.Format {
	case "yaml":
	default:
		req.Format = "json"
	}

	resp.Config.Write(buf, req.Format)

	contentType := "application/json"
	if req.Format == "yaml" {
		contentType = "text/x-yaml"
	}

	ctx.Data(http.StatusOK, contentType, buf.Bytes())
}
