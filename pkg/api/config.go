package api

import (
	"bytes"
	"fmt"
	"net/http"
	"os"

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
	//     Security:
	//       basicAuth: []
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

// swagger:parameters saveConfigRequest
type saveConfigRequest struct {
	// output format, one of yaml|json, default is yaml.
	// in: query
	Format string `form:"format" json:"format"`
}

// successful operation.
// swagger:response saveConfigResponse
type saveConfigResponse struct {
	Data Response
}

func saveConfig(ctx *gin.Context) {
	// swagger:route POST /config ConfigManagement saveConfigRequest
	//
	// Save current config to file (gost.yaml or gost.json).
	//
	//     Security:
	//       basicAuth: []
	//
	//     Responses:
	//       200: saveConfigResponse

	var req saveConfigRequest
	ctx.ShouldBindQuery(&req)

	file := "gost.yaml"
	switch req.Format {
	case "json":
		file = "gost.json"
	default:
		req.Format = "yaml"
	}

	f, err := os.Create(file)
	if err != nil {
		writeError(ctx, &Error{
			statusCode: http.StatusInternalServerError,
			Code:       40005,
			Msg:        fmt.Sprintf("create file: %s", err.Error()),
		})
		return
	}
	defer f.Close()

	if err := config.Global().Write(f, req.Format); err != nil {
		writeError(ctx, &Error{
			statusCode: http.StatusInternalServerError,
			Code:       40006,
			Msg:        fmt.Sprintf("write: %s", err.Error()),
		})
		return
	}

	ctx.JSON(http.StatusOK, Response{
		Msg: "OK",
	})
}
