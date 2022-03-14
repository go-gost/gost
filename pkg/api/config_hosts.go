package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-gost/gost/v3/pkg/config"
	"github.com/go-gost/gost/v3/pkg/config/parsing"
	"github.com/go-gost/gost/v3/pkg/registry"
)

// swagger:parameters createHostsRequest
type createHostsRequest struct {
	// in: body
	Data config.HostsConfig `json:"data"`
}

// successful operation.
// swagger:response createHostsResponse
type createHostsesponse struct {
	Data Response
}

func createHosts(ctx *gin.Context) {
	// swagger:route POST /config/hosts ConfigManagement createHostsRequest
	//
	// Create a new hosts, the name of the hosts must be unique in hosts list.
	//
	//     Security:
	//       basicAuth: []
	//
	//     Responses:
	//       200: createHostsResponse

	var req createHostsRequest
	ctx.ShouldBindJSON(&req.Data)

	if req.Data.Name == "" {
		writeError(ctx, ErrInvalid)
		return
	}

	v := parsing.ParseHosts(&req.Data)

	if err := registry.HostsRegistry().Register(req.Data.Name, v); err != nil {
		writeError(ctx, ErrDup)
		return
	}

	cfg := config.Global()
	cfg.Hosts = append(cfg.Hosts, &req.Data)
	config.SetGlobal(cfg)

	ctx.JSON(http.StatusOK, Response{
		Msg: "OK",
	})
}

// swagger:parameters updateHostsRequest
type updateHostsRequest struct {
	// in: path
	// required: true
	Hosts string `uri:"hosts" json:"hosts"`
	// in: body
	Data config.HostsConfig `json:"data"`
}

// successful operation.
// swagger:response updateHostsResponse
type updateHostsResponse struct {
	Data Response
}

func updateHosts(ctx *gin.Context) {
	// swagger:route PUT /config/hosts/{hosts} ConfigManagement updateHostsRequest
	//
	// Update hosts by name, the hosts must already exist.
	//
	//     Security:
	//       basicAuth: []
	//
	//     Responses:
	//       200: updateHostsResponse

	var req updateHostsRequest
	ctx.ShouldBindUri(&req)
	ctx.ShouldBindJSON(&req.Data)

	if !registry.HostsRegistry().IsRegistered(req.Hosts) {
		writeError(ctx, ErrNotFound)
		return
	}

	req.Data.Name = req.Hosts

	v := parsing.ParseHosts(&req.Data)

	registry.HostsRegistry().Unregister(req.Hosts)

	if err := registry.HostsRegistry().Register(req.Hosts, v); err != nil {
		writeError(ctx, ErrDup)
		return
	}

	cfg := config.Global()
	for i := range cfg.Hosts {
		if cfg.Hosts[i].Name == req.Hosts {
			cfg.Hosts[i] = &req.Data
			break
		}
	}
	config.SetGlobal(cfg)

	ctx.JSON(http.StatusOK, Response{
		Msg: "OK",
	})
}

// swagger:parameters deleteHostsRequest
type deleteHostsRequest struct {
	// in: path
	// required: true
	Hosts string `uri:"hosts" json:"hosts"`
}

// successful operation.
// swagger:response deleteHostsResponse
type deleteHostsResponse struct {
	Data Response
}

func deleteHosts(ctx *gin.Context) {
	// swagger:route DELETE /config/hosts/{hosts} ConfigManagement deleteHostsRequest
	//
	// Delete hosts by name.
	//
	//     Security:
	//       basicAuth: []
	//
	//     Responses:
	//       200: deleteHostsResponse

	var req deleteHostsRequest
	ctx.ShouldBindUri(&req)

	if !registry.HostsRegistry().IsRegistered(req.Hosts) {
		writeError(ctx, ErrNotFound)
		return
	}
	registry.HostsRegistry().Unregister(req.Hosts)

	cfg := config.Global()
	hosts := cfg.Hosts
	cfg.Hosts = nil
	for _, s := range hosts {
		if s.Name == req.Hosts {
			continue
		}
		cfg.Hosts = append(cfg.Hosts, s)
	}
	config.SetGlobal(cfg)

	ctx.JSON(http.StatusOK, Response{
		Msg: "OK",
	})
}
