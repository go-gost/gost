package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-gost/gost/pkg/config"
	"github.com/go-gost/gost/pkg/config/parsing"
	"github.com/go-gost/gost/pkg/registry"
)

// swagger:parameters createBypassRequest
type createBypassRequest struct {
	// in: body
	Data config.BypassConfig `json:"data"`
}

// successful operation.
// swagger:response createBypassResponse
type createBypassResponse struct {
	Data Response
}

func createBypass(ctx *gin.Context) {
	// swagger:route POST /config/bypasses ConfigManagement createBypassRequest
	//
	// Create a new bypass, the name of bypass must be unique in bypass list.
	//
	//     Security:
	//       basicAuth: []
	//
	//     Responses:
	//       200: createBypassResponse

	var req createBypassRequest
	ctx.ShouldBindJSON(&req.Data)

	if req.Data.Name == "" {
		writeError(ctx, ErrInvalid)
		return
	}

	v := parsing.ParseBypass(&req.Data)

	if err := registry.Bypass().Register(req.Data.Name, v); err != nil {
		writeError(ctx, ErrDup)
		return
	}

	cfg := config.Global()
	cfg.Bypasses = append(cfg.Bypasses, &req.Data)
	config.SetGlobal(cfg)

	ctx.JSON(http.StatusOK, Response{
		Msg: "OK",
	})
}

// swagger:parameters updateBypassRequest
type updateBypassRequest struct {
	// in: path
	// required: true
	Bypass string `uri:"bypass" json:"bypass"`
	// in: body
	Data config.BypassConfig `json:"data"`
}

// successful operation.
// swagger:response updateBypassResponse
type updateBypassResponse struct {
	Data Response
}

func updateBypass(ctx *gin.Context) {
	// swagger:route PUT /config/bypasses/{bypass} ConfigManagement updateBypassRequest
	//
	// Update bypass by name, the bypass must already exist.
	//
	//     Security:
	//       basicAuth: []
	//
	//     Responses:
	//       200: updateBypassResponse

	var req updateBypassRequest
	ctx.ShouldBindUri(&req)
	ctx.ShouldBindJSON(&req.Data)

	if !registry.Bypass().IsRegistered(req.Bypass) {
		writeError(ctx, ErrNotFound)
		return
	}

	req.Data.Name = req.Bypass

	v := parsing.ParseBypass(&req.Data)

	registry.Bypass().Unregister(req.Bypass)

	if err := registry.Bypass().Register(req.Bypass, v); err != nil {
		writeError(ctx, ErrDup)
		return
	}

	cfg := config.Global()
	for i := range cfg.Bypasses {
		if cfg.Bypasses[i].Name == req.Bypass {
			cfg.Bypasses[i] = &req.Data
			break
		}
	}
	config.SetGlobal(cfg)

	ctx.JSON(http.StatusOK, Response{
		Msg: "OK",
	})
}

// swagger:parameters deleteBypassRequest
type deleteBypassRequest struct {
	// in: path
	// required: true
	Bypass string `uri:"bypass" json:"bypass"`
}

// successful operation.
// swagger:response deleteBypassResponse
type deleteBypassResponse struct {
	Data Response
}

func deleteBypass(ctx *gin.Context) {
	// swagger:route DELETE /config/bypasses/{bypass} ConfigManagement deleteBypassRequest
	//
	// Delete bypass by name.
	//
	//     Security:
	//       basicAuth: []
	//
	//     Responses:
	//       200: deleteBypassResponse

	var req deleteBypassRequest
	ctx.ShouldBindUri(&req)

	if !registry.Bypass().IsRegistered(req.Bypass) {
		writeError(ctx, ErrNotFound)
		return
	}
	registry.Bypass().Unregister(req.Bypass)

	cfg := config.Global()
	bypasses := cfg.Bypasses
	cfg.Bypasses = nil
	for _, s := range bypasses {
		if s.Name == req.Bypass {
			continue
		}
		cfg.Bypasses = append(cfg.Bypasses, s)
	}
	config.SetGlobal(cfg)

	ctx.JSON(http.StatusOK, Response{
		Msg: "OK",
	})
}
