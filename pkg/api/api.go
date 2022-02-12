package api

import (
	"embed"
)

var (
	//go:embed swagger.yaml
	swaggerDoc embed.FS
)

type Response struct {
	Code int    `json:"code,omitempty"`
	Msg  string `json:"msg,omitempty"`
}
