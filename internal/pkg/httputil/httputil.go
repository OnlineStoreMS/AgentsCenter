package httputil

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"agentscenter/internal/pkg/response"
	"agentscenter/internal/service"

	"github.com/gin-gonic/gin"
)

func ParseID(c *gin.Context) (uint64, error) {
	return ParseNamedID(c, "id")
}

func ParseNamedID(c *gin.Context, param string) (uint64, error) {
	idStr := c.Param(param)
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id == 0 {
		return 0, errors.New("invalid id")
	}
	return id, nil
}

func ParsePage(c *gin.Context) (page, pageSize int) {
	page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ = strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 20
	}
	return page, pageSize
}

func HandleServiceError(c *gin.Context, err error) {
	msg := err.Error()
	switch {
	case errors.Is(err, service.ErrNotFound):
		response.Fail(c, http.StatusNotFound, msg)
	case errors.Is(err, service.ErrBadRequest):
		response.Fail(c, http.StatusBadRequest, msg)
	case errors.Is(err, service.ErrAgentAuth):
		response.Fail(c, http.StatusUnauthorized, msg)
	case errors.Is(err, service.ErrJobState):
		response.Fail(c, http.StatusConflict, msg)
	case errors.Is(err, service.ErrNoAgentOnline):
		response.Fail(c, http.StatusConflict, msg)
	case strings.Contains(msg, "参数"):
		response.Fail(c, http.StatusBadRequest, msg)
	default:
		response.Fail(c, http.StatusInternalServerError, msg)
	}
}
