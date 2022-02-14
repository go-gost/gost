package api

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
)

var (
	ErrInvalid  = &Error{statusCode: http.StatusBadRequest, Code: 40001, Msg: "instance invalid"}
	ErrDup      = &Error{statusCode: http.StatusBadRequest, Code: 40002, Msg: "instance duplicated"}
	ErrCreate   = &Error{statusCode: http.StatusConflict, Code: 40003, Msg: "instance creation failed"}
	ErrNotFound = &Error{statusCode: http.StatusBadRequest, Code: 40004, Msg: "instance not found"}
	ErrSave     = &Error{statusCode: http.StatusInternalServerError, Code: 40005, Msg: "save config failed"}
)

// Error is an api error.
type Error struct {
	statusCode int
	Code       int    `json:"code"`
	Msg        string `json:"msg"`
}

func (e *Error) Error() string {
	b, _ := json.Marshal(e)
	return string(b)
}

func writeError(c *gin.Context, err error) {
	// c.Set(HTTPResponseTag, err)
	c.JSON(getStatusCode(err), err)
}

func getStatusCode(err error) int {
	if err == nil {
		return http.StatusOK
	}
	if e, ok := err.(*Error); ok {
		if e.statusCode >= http.StatusOK && e.statusCode < 600 {
			return e.statusCode
		}
	}
	return http.StatusInternalServerError
}
