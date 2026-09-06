package agentapi

import (
	"net/http"
	"strconv"
	"strings"

	"agentscenter/internal/dto"
	"agentscenter/internal/model"
	"agentscenter/internal/pkg/httputil"
	"agentscenter/internal/pkg/response"
	"agentscenter/internal/service"

	"github.com/gin-gonic/gin"
)

const contextAgent = "agent_device"

type Handler struct {
	svc *service.AgentService
}

func NewHandler(svc *service.AgentService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(c *gin.Context) {
	var in dto.AgentRegisterInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.svc.Register(1, &in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *Handler) AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := strings.TrimSpace(c.GetHeader("X-Agent-Key"))
		secret := strings.TrimSpace(c.GetHeader("X-Agent-Secret"))
		agent, err := h.svc.Authenticate(key, secret)
		if err != nil {
			httputil.HandleServiceError(c, err)
			c.Abort()
			return
		}
		c.Set(contextAgent, agent)
		c.Next()
	}
}

func (h *Handler) Heartbeat(c *gin.Context) {
	agent := mustAgent(c)
	if agent == nil {
		response.Fail(c, http.StatusUnauthorized, service.ErrAgentAuth.Error())
		return
	}
	var in dto.AgentHeartbeatInput
	_ = c.ShouldBindJSON(&in)
	item, err := h.svc.Heartbeat(agent, &in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *Handler) ClaimJobs(c *gin.Context) {
	agent := mustAgent(c)
	if agent == nil {
		response.Fail(c, http.StatusUnauthorized, service.ErrAgentAuth.Error())
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "1"))
	items, err := h.svc.ClaimJobs(agent, limit)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, items)
}

func (h *Handler) ReportJob(c *gin.Context) {
	agent := mustAgent(c)
	if agent == nil {
		response.Fail(c, http.StatusUnauthorized, service.ErrAgentAuth.Error())
		return
	}
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	var in dto.JobReportInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.svc.ReportJob(agent, id, &in); err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, gin.H{"ok": true})
}

func mustAgent(c *gin.Context) *model.Agent {
	v, ok := c.Get(contextAgent)
	if !ok {
		return nil
	}
	agent, _ := v.(*model.Agent)
	return agent
}
