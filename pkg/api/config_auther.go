package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-gost/gost/pkg/config"
	"github.com/go-gost/gost/pkg/config/parsing"
	"github.com/go-gost/gost/pkg/registry"
)

// swagger:parameters createAutherRequest
type createAutherRequest struct {
	// in: body
	Data config.AutherConfig `json:"data"`
}

// successful operation.
// swagger:response createAutherResponse
type createAutherResponse struct {
	Data Response
}

func createAuther(ctx *gin.Context) {
	// swagger:route POST /config/authers ConfigManagement createAutherRequest
	//
	// Create a new auther, the name of the auther must be unique in auther list.
	//
	//     Security:
	//       basicAuth: []
	//
	//     Responses:
	//       200: createAutherResponse

	var req createAutherRequest
	ctx.ShouldBindJSON(&req.Data)

	if req.Data.Name == "" {
		writeError(ctx, ErrInvalid)
		return
	}

	v := parsing.ParseAuther(&req.Data)
	if err := registry.Auther().Register(req.Data.Name, v); err != nil {
		writeError(ctx, ErrDup)
		return
	}

	cfg := config.Global()
	cfg.Authers = append(cfg.Authers, &req.Data)
	config.SetGlobal(cfg)

	ctx.JSON(http.StatusOK, Response{
		Msg: "OK",
	})
}

// swagger:parameters updateAutherRequest
type updateAutherRequest struct {
	// in: path
	// required: true
	Auther string `uri:"auther" json:"auther"`
	// in: body
	Data config.AutherConfig `json:"data"`
}

// successful operation.
// swagger:response updateAutherResponse
type updateAutherResponse struct {
	Data Response
}

func updateAuther(ctx *gin.Context) {
	// swagger:route PUT /config/authers/{auther} ConfigManagement updateAutherRequest
	//
	// Update auther by name, the auther must already exist.
	//
	//     Security:
	//       basicAuth: []
	//
	//     Responses:
	//       200: updateAutherResponse

	var req updateAutherRequest
	ctx.ShouldBindUri(&req)
	ctx.ShouldBindJSON(&req.Data)

	if !registry.Auther().IsRegistered(req.Auther) {
		writeError(ctx, ErrNotFound)
		return
	}

	req.Data.Name = req.Auther

	v := parsing.ParseAuther(&req.Data)
	registry.Auther().Unregister(req.Auther)

	if err := registry.Auther().Register(req.Auther, v); err != nil {
		writeError(ctx, ErrDup)
		return
	}

	cfg := config.Global()
	for i := range cfg.Authers {
		if cfg.Authers[i].Name == req.Auther {
			cfg.Authers[i] = &req.Data
			break
		}
	}
	config.SetGlobal(cfg)

	ctx.JSON(http.StatusOK, Response{
		Msg: "OK",
	})
}

// swagger:parameters deleteAutherRequest
type deleteAutherRequest struct {
	// in: path
	// required: true
	Auther string `uri:"auther" json:"auther"`
}

// successful operation.
// swagger:response deleteAutherResponse
type deleteAutherResponse struct {
	Data Response
}

func deleteAuther(ctx *gin.Context) {
	// swagger:route DELETE /config/authers/{auther} ConfigManagement deleteAutherRequest
	//
	// Delete auther by name.
	//
	//     Security:
	//       basicAuth: []
	//
	//     Responses:
	//       200: deleteAutherResponse

	var req deleteAutherRequest
	ctx.ShouldBindUri(&req)

	if !registry.Auther().IsRegistered(req.Auther) {
		writeError(ctx, ErrNotFound)
		return
	}
	registry.Auther().Unregister(req.Auther)

	cfg := config.Global()
	authers := cfg.Authers
	cfg.Authers = nil
	for _, s := range authers {
		if s.Name == req.Auther {
			continue
		}
		cfg.Authers = append(cfg.Authers, s)
	}
	config.SetGlobal(cfg)

	ctx.JSON(http.StatusOK, Response{
		Msg: "OK",
	})
}
