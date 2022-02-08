package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-gost/gost/pkg/config"
	"github.com/go-gost/gost/pkg/config/parsing"
	"github.com/go-gost/gost/pkg/registry"
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
	// create a new hosts, the name of the hosts must be unique in hosts list.
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

	if err := registry.Hosts().Register(req.Data.Name, v); err != nil {
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
	// update hosts by name, the hosts must already exist.
	//
	//     Responses:
	//       200: updateHostsResponse

	var req updateHostsRequest
	ctx.ShouldBindUri(&req)
	ctx.ShouldBindJSON(&req.Data)

	if !registry.Hosts().IsRegistered(req.Hosts) {
		writeError(ctx, ErrNotFound)
		return
	}

	req.Data.Name = req.Hosts

	v := parsing.ParseHosts(&req.Data)

	registry.Hosts().Unregister(req.Hosts)

	if err := registry.Hosts().Register(req.Hosts, v); err != nil {
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
	// delete hosts by name.
	//
	//     Responses:
	//       200: deleteHostsResponse

	var req deleteHostsRequest
	ctx.ShouldBindUri(&req)

	svc := registry.Hosts().Get(req.Hosts)
	if svc == nil {
		writeError(ctx, ErrNotFound)
		return
	}
	registry.Hosts().Unregister(req.Hosts)

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
