package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-gost/gost/pkg/config"
	"github.com/go-gost/gost/pkg/config/parsing"
	"github.com/go-gost/gost/pkg/registry"
)

// swagger:parameters createAdmissionRequest
type createAdmissionRequest struct {
	// in: body
	Data config.AdmissionConfig `json:"data"`
}

// successful operation.
// swagger:response createAdmissionResponse
type createAdmissionResponse struct {
	Data Response
}

func createAdmission(ctx *gin.Context) {
	// swagger:route POST /config/admissions ConfigManagement createAdmissionRequest
	//
	// Create a new admission, the name of admission must be unique in admission list.
	//
	//     Security:
	//       basicAuth: []
	//
	//     Responses:
	//       200: createAdmissionResponse

	var req createAdmissionRequest
	ctx.ShouldBindJSON(&req.Data)

	if req.Data.Name == "" {
		writeError(ctx, ErrInvalid)
		return
	}

	v := parsing.ParseAdmission(&req.Data)

	if err := registry.AdmissionRegistry().Register(req.Data.Name, v); err != nil {
		writeError(ctx, ErrDup)
		return
	}

	cfg := config.Global()
	cfg.Admissions = append(cfg.Admissions, &req.Data)
	config.SetGlobal(cfg)

	ctx.JSON(http.StatusOK, Response{
		Msg: "OK",
	})
}

// swagger:parameters updateAdmissionRequest
type updateAdmissionRequest struct {
	// in: path
	// required: true
	Admission string `uri:"admission" json:"admission"`
	// in: body
	Data config.AdmissionConfig `json:"data"`
}

// successful operation.
// swagger:response updateAdmissionResponse
type updateAdmissionResponse struct {
	Data Response
}

func updateAdmission(ctx *gin.Context) {
	// swagger:route PUT /config/admissions/{admission} ConfigManagement updateAdmissionRequest
	//
	// Update admission by name, the admission must already exist.
	//
	//     Security:
	//       basicAuth: []
	//
	//     Responses:
	//       200: updateAdmissionResponse

	var req updateAdmissionRequest
	ctx.ShouldBindUri(&req)
	ctx.ShouldBindJSON(&req.Data)

	if !registry.AdmissionRegistry().IsRegistered(req.Admission) {
		writeError(ctx, ErrNotFound)
		return
	}

	req.Data.Name = req.Admission

	v := parsing.ParseAdmission(&req.Data)

	registry.AdmissionRegistry().Unregister(req.Admission)

	if err := registry.AdmissionRegistry().Register(req.Admission, v); err != nil {
		writeError(ctx, ErrDup)
		return
	}

	cfg := config.Global()
	for i := range cfg.Admissions {
		if cfg.Admissions[i].Name == req.Admission {
			cfg.Admissions[i] = &req.Data
			break
		}
	}
	config.SetGlobal(cfg)

	ctx.JSON(http.StatusOK, Response{
		Msg: "OK",
	})
}

// swagger:parameters deleteAdmissionRequest
type deleteAdmissionRequest struct {
	// in: path
	// required: true
	Admission string `uri:"admission" json:"admission"`
}

// successful operation.
// swagger:response deleteAdmissionResponse
type deleteAdmissionResponse struct {
	Data Response
}

func deleteAdmission(ctx *gin.Context) {
	// swagger:route DELETE /config/admissions/{admission} ConfigManagement deleteAdmissionRequest
	//
	// Delete admission by name.
	//
	//     Security:
	//       basicAuth: []
	//
	//     Responses:
	//       200: deleteAdmissionResponse

	var req deleteAdmissionRequest
	ctx.ShouldBindUri(&req)

	if !registry.AdmissionRegistry().IsRegistered(req.Admission) {
		writeError(ctx, ErrNotFound)
		return
	}
	registry.AdmissionRegistry().Unregister(req.Admission)

	cfg := config.Global()
	admissiones := cfg.Admissions
	cfg.Admissions = nil
	for _, s := range admissiones {
		if s.Name == req.Admission {
			continue
		}
		cfg.Admissions = append(cfg.Admissions, s)
	}
	config.SetGlobal(cfg)

	ctx.JSON(http.StatusOK, Response{
		Msg: "OK",
	})
}
