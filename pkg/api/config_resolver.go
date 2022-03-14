package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-gost/gost/v3/pkg/config"
	"github.com/go-gost/gost/v3/pkg/config/parsing"
	"github.com/go-gost/gost/v3/pkg/registry"
)

// swagger:parameters createResolverRequest
type createResolverRequest struct {
	// in: body
	Data config.ResolverConfig `json:"data"`
}

// successful operation.
// swagger:response createResolverResponse
type createResolverResponse struct {
	Data Response
}

func createResolver(ctx *gin.Context) {
	// swagger:route POST /config/resolvers ConfigManagement createResolverRequest
	//
	// Create a new resolver, the name of the resolver must be unique in resolver list.
	//
	//     Security:
	//       basicAuth: []
	//
	//     Responses:
	//       200: createResolverResponse

	var req createResolverRequest
	ctx.ShouldBindJSON(&req.Data)

	if req.Data.Name == "" {
		writeError(ctx, ErrInvalid)
		return
	}

	v, err := parsing.ParseResolver(&req.Data)
	if err != nil {
		writeError(ctx, ErrCreate)
		return
	}

	if err := registry.ResolverRegistry().Register(req.Data.Name, v); err != nil {
		writeError(ctx, ErrDup)
		return
	}

	cfg := config.Global()
	cfg.Resolvers = append(cfg.Resolvers, &req.Data)
	config.SetGlobal(cfg)

	ctx.JSON(http.StatusOK, Response{
		Msg: "OK",
	})
}

// swagger:parameters updateResolverRequest
type updateResolverRequest struct {
	// in: path
	// required: true
	Resolver string `uri:"resolver" json:"resolver"`
	// in: body
	Data config.ResolverConfig `json:"data"`
}

// successful operation.
// swagger:response updateResolverResponse
type updateResolverResponse struct {
	Data Response
}

func updateResolver(ctx *gin.Context) {
	// swagger:route PUT /config/resolvers/{resolver} ConfigManagement updateResolverRequest
	//
	// Update resolver by name, the resolver must already exist.
	//
	//     Security:
	//       basicAuth: []
	//
	//     Responses:
	//       200: updateResolverResponse

	var req updateResolverRequest
	ctx.ShouldBindUri(&req)
	ctx.ShouldBindJSON(&req.Data)

	if !registry.ResolverRegistry().IsRegistered(req.Resolver) {
		writeError(ctx, ErrNotFound)
		return
	}

	req.Data.Name = req.Resolver

	v, err := parsing.ParseResolver(&req.Data)
	if err != nil {
		writeError(ctx, ErrCreate)
		return
	}

	registry.ResolverRegistry().Unregister(req.Resolver)

	if err := registry.ResolverRegistry().Register(req.Resolver, v); err != nil {
		writeError(ctx, ErrDup)
		return
	}

	cfg := config.Global()
	for i := range cfg.Resolvers {
		if cfg.Resolvers[i].Name == req.Resolver {
			cfg.Resolvers[i] = &req.Data
			break
		}
	}
	config.SetGlobal(cfg)

	ctx.JSON(http.StatusOK, Response{
		Msg: "OK",
	})
}

// swagger:parameters deleteResolverRequest
type deleteResolverRequest struct {
	// in: path
	// required: true
	Resolver string `uri:"resolver" json:"resolver"`
}

// successful operation.
// swagger:response deleteResolverResponse
type deleteResolverResponse struct {
	Data Response
}

func deleteResolver(ctx *gin.Context) {
	// swagger:route DELETE /config/resolvers/{resolver} ConfigManagement deleteResolverRequest
	//
	// Delete resolver by name.
	//
	//     Security:
	//       basicAuth: []
	//
	//     Responses:
	//       200: deleteResolverResponse

	var req deleteResolverRequest
	ctx.ShouldBindUri(&req)

	if !registry.ResolverRegistry().IsRegistered(req.Resolver) {
		writeError(ctx, ErrNotFound)
		return
	}
	registry.ResolverRegistry().Unregister(req.Resolver)

	cfg := config.Global()
	resolvers := cfg.Resolvers
	cfg.Resolvers = nil
	for _, s := range resolvers {
		if s.Name == req.Resolver {
			continue
		}
		cfg.Resolvers = append(cfg.Resolvers, s)
	}
	config.SetGlobal(cfg)

	ctx.JSON(http.StatusOK, Response{
		Msg: "OK",
	})
}
