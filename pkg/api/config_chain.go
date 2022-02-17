package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-gost/gost/pkg/config"
	"github.com/go-gost/gost/pkg/config/parsing"
	"github.com/go-gost/gost/pkg/registry"
)

// swagger:parameters createChainRequest
type createChainRequest struct {
	// in: body
	Data config.ChainConfig `json:"data"`
}

// successful operation.
// swagger:response createChainResponse
type createChainResponse struct {
	Data Response
}

func createChain(ctx *gin.Context) {
	// swagger:route POST /config/chains ConfigManagement createChainRequest
	//
	// Create a new chain, the name of chain must be unique in chain list.
	//
	//     Security:
	//       basicAuth: []
	//
	//     Responses:
	//       200: createChainResponse

	var req createChainRequest
	ctx.ShouldBindJSON(&req.Data)

	if req.Data.Name == "" {
		writeError(ctx, ErrInvalid)
		return
	}

	v, err := parsing.ParseChain(&req.Data)
	if err != nil {
		writeError(ctx, ErrCreate)
		return
	}

	if err := registry.Chain().Register(req.Data.Name, v); err != nil {
		writeError(ctx, ErrDup)
		return
	}

	cfg := config.Global()
	cfg.Chains = append(cfg.Chains, &req.Data)
	config.SetGlobal(cfg)

	ctx.JSON(http.StatusOK, Response{
		Msg: "OK",
	})
}

// swagger:parameters updateChainRequest
type updateChainRequest struct {
	// in: path
	// required: true
	// chain name
	Chain string `uri:"chain" json:"chain"`
	// in: body
	Data config.ChainConfig `json:"data"`
}

// successful operation.
// swagger:response updateChainResponse
type updateChainResponse struct {
	Data Response
}

func updateChain(ctx *gin.Context) {
	// swagger:route PUT /config/chains/{chain} ConfigManagement updateChainRequest
	//
	// Update chain by name, the chain must already exist.
	//
	//     Security:
	//       basicAuth: []
	//
	//     Responses:
	//       200: updateChainResponse

	var req updateChainRequest
	ctx.ShouldBindUri(&req)
	ctx.ShouldBindJSON(&req.Data)

	if !registry.Chain().IsRegistered(req.Chain) {
		writeError(ctx, ErrNotFound)
		return
	}

	req.Data.Name = req.Chain

	v, err := parsing.ParseChain(&req.Data)
	if err != nil {
		writeError(ctx, ErrCreate)
		return
	}

	registry.Chain().Unregister(req.Chain)

	if err := registry.Chain().Register(req.Chain, v); err != nil {
		writeError(ctx, ErrDup)
		return
	}

	cfg := config.Global()
	for i := range cfg.Chains {
		if cfg.Chains[i].Name == req.Chain {
			cfg.Chains[i] = &req.Data
			break
		}
	}
	config.SetGlobal(cfg)

	ctx.JSON(http.StatusOK, Response{
		Msg: "OK",
	})
}

// swagger:parameters deleteChainRequest
type deleteChainRequest struct {
	// in: path
	// required: true
	Chain string `uri:"chain" json:"chain"`
}

// successful operation.
// swagger:response deleteChainResponse
type deleteChainResponse struct {
	Data Response
}

func deleteChain(ctx *gin.Context) {
	// swagger:route DELETE /config/chains/{chain} ConfigManagement deleteChainRequest
	//
	// Delete chain by name.
	//
	//     Security:
	//       basicAuth: []
	//
	//     Responses:
	//       200: deleteChainResponse

	var req deleteChainRequest
	ctx.ShouldBindUri(&req)

	if !registry.Chain().IsRegistered(req.Chain) {
		writeError(ctx, ErrNotFound)
		return
	}
	registry.Chain().Unregister(req.Chain)

	cfg := config.Global()
	chains := cfg.Chains
	cfg.Chains = nil
	for _, s := range chains {
		if s.Name == req.Chain {
			continue
		}
		cfg.Chains = append(cfg.Chains, s)
	}
	config.SetGlobal(cfg)

	ctx.JSON(http.StatusOK, Response{
		Msg: "OK",
	})
}
