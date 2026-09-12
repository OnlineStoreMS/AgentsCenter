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
		TargetAgentID:    in.TargetAgentID,
	})
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.Created(c, job)
}

func (h *Handler) GetJobs(c *gin.Context) {
	tenantID, _ := strconv.ParseUint(c.Query("tenantId"), 10, 64)
	raw := strings.TrimSpace(c.Query("ids"))
	if raw == "" {
		response.OK(c, gin.H{"list": []any{}})
		return
	}
	parts := strings.Split(raw, ",")
	ids := make([]uint64, 0, len(parts))
	for _, p := range parts {
		id, err := strconv.ParseUint(strings.TrimSpace(p), 10, 64)
		if err != nil || id == 0 {
			continue
		}
		ids = append(ids, id)
	}
	list, err := h.svc.GetJobsByIDs(tenantID, ids)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, gin.H{"list": list})
}

func (h *Handler) ListAgents(c *gin.Context) {
	tenantID, _ := strconv.ParseUint(c.Query("tenantId"), 10, 64)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "100"))
	onlineOnly := c.Query("onlineOnly") == "1" || c.Query("onlineOnly") == "true"
	skill := strings.TrimSpace(c.Query("skill"))
	list, total, err := h.svc.ListAgentsFiltered(tenantID, page, pageSize, onlineOnly, skill)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
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

func (h *Handler) UpsertAssignment(c *gin.Context) {
	var in dto.InternalUpsertAssignmentInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	row, job, err := h.svc.UpsertAssignment(in.TenantID, 0, &dto.UpsertAssignmentInput{
		JobType:          in.JobType,
		Platform:         in.Platform,
		PlatformShopID:   in.PlatformShopID,
		PlatformShopName: in.PlatformShopName,
		Enabled:          in.Enabled,
		RunPolicy:        in.RunPolicy,
		IntervalMinutes:  in.IntervalMinutes,
		TriggerNow:       in.TriggerNow,
		ParamsJSON:       in.ParamsJSON,
		Source:           in.Source,
		Priority:         in.Priority,
	})
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.Created(c, gin.H{"assignment": row, "job": job})
}

// TriggerByShop 按店铺+技能立即创建执行单（会先确保订阅存在）。
func (h *Handler) TriggerByShop(c *gin.Context) {
	var in dto.InternalUpsertAssignmentInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	in.TriggerNow = true
	if in.Enabled == nil {
		t := true
		in.Enabled = &t
	}
	row, job, err := h.svc.UpsertAssignment(in.TenantID, 0, &dto.UpsertAssignmentInput{
		JobType:          in.JobType,
		Platform:         in.Platform,
		PlatformShopID:   in.PlatformShopID,
		PlatformShopName: in.PlatformShopName,
		Enabled:          in.Enabled,
		RunPolicy:        in.RunPolicy,
		IntervalMinutes:  in.IntervalMinutes,
		TriggerNow:       true,
		ParamsJSON:       in.ParamsJSON,
		Source:           in.Source,
		Priority:         in.Priority,
	})
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, gin.H{"assignment": row, "job": job})
}
