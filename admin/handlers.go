package admin

import (
	"net/http"

	"agentscenter/internal/dto"
	"agentscenter/internal/pkg/authcontext"
	"agentscenter/internal/pkg/httputil"
	"agentscenter/internal/pkg/response"
	"agentscenter/internal/service"

	"github.com/gin-gonic/gin"
)

type Handlers struct {
	svc *service.AgentService
}

func NewHandlers(svc *service.AgentService) *Handlers {
	return &Handlers{svc: svc}
}

func RegisterRoutes(g *gin.RouterGroup, h *Handlers) {
	g.GET("/dashboard/stats", h.DashboardStats)
	g.GET("/skills", h.ListSkills)

	g.GET("/agents", h.ListAgents)
	g.GET("/shops", h.ListShops)

	g.GET("/jobs", h.ListJobs)
	g.POST("/jobs", h.CreateJob)
}

func (h *Handlers) DashboardStats(c *gin.Context) {
	tenantID := authcontext.TenantID(c)
	_, agentTotal, err := h.svc.ListAgents(tenantID, 1, 500)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	onlineList, _, _ := h.svc.ListAgents(tenantID, 1, 500)
	var online int64
	for _, a := range onlineList {
		if a.Status == "online" {
			online++
		}
	}
	_, shopTotal, _ := h.svc.ListShops(tenantID, 1, 1, "")
	_, pendingTotal, _ := h.svc.ListJobs(tenantID, 1, 1, "pending", "")
	_, runningTotal, _ := h.svc.ListJobs(tenantID, 1, 1, "running", "")
	response.OK(c, gin.H{
		"agentTotal":  agentTotal,
		"agentOnline": online,
		"shopTotal":   shopTotal,
		"jobPending":  pendingTotal,
		"jobRunning":  runningTotal,
	})
}

func (h *Handlers) ListSkills(c *gin.Context) {
	response.OK(c, h.svc.SkillCatalog())
}

func (h *Handlers) ListAgents(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	list, total, err := h.svc.ListAgents(authcontext.TenantID(c), page, pageSize)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
}

func (h *Handlers) ListShops(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	list, total, err := h.svc.ListShops(authcontext.TenantID(c), page, pageSize, c.Query("platform"))
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
}

func (h *Handlers) ListJobs(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	list, total, err := h.svc.ListJobs(authcontext.TenantID(c), page, pageSize, c.Query("status"), c.Query("jobType"))
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
}

func (h *Handlers) CreateJob(c *gin.Context) {
	var in dto.CreateJobInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	job, err := h.svc.CreateJob(authcontext.TenantID(c), authcontext.UserID(c), &in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.Created(c, job)
}
