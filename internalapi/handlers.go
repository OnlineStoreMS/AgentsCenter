package internalapi

import (
	"net/http"
	"strconv"
	"strings"

	"agentscenter/internal/dto"
	"agentscenter/internal/pkg/httputil"
	"agentscenter/internal/pkg/response"
	"agentscenter/internal/service"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc   *service.AgentService
	token string
}

func NewHandler(svc *service.AgentService, token string) *Handler {
	return &Handler{svc: svc, token: strings.TrimSpace(token)}
}

func (h *Handler) AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		got := strings.TrimSpace(c.GetHeader("X-Internal-Token"))
		if h.token == "" || got == "" || got != h.token {
			response.Fail(c, http.StatusUnauthorized, "invalid internal token")
			c.Abort()
			return
		}
		c.Next()
	}
}

func (h *Handler) CreateJob(c *gin.Context) {
	var in dto.InternalCreateJobInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	job, err := h.svc.CreateJob(in.TenantID, 0, &dto.CreateJobInput{
		JobType:          in.JobType,
		Platform:         in.Platform,
		PlatformShopID:   in.PlatformShopID,
		PlatformShopName: in.PlatformShopName,
		ParamsJSON:       in.ParamsJSON,
		Source:           in.Source,
		Priority:         in.Priority,
	})
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.Created(c, job)
}

func (h *Handler) ListShops(c *gin.Context) {
	tenantID, _ := strconv.ParseUint(c.Query("tenantId"), 10, 64)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "100"))
	onlineOnly := c.Query("onlineOnly") != "0" && c.Query("onlineOnly") != "false"
	list, total, err := h.svc.ListShops(tenantID, page, pageSize, c.Query("platform"), onlineOnly)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
}
