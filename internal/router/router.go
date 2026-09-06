package router

import (
	"agentscenter/admin"
	adminmw "agentscenter/admin/middleware"
	"agentscenter/agentapi"
	"agentscenter/internal/config"
	jwtmgr "agentscenter/internal/pkg/jwt"
	"agentscenter/internal/repo"
	"agentscenter/internal/scheduler"
	"agentscenter/internal/service"
	"agentscenter/internalapi"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func Setup(db *gorm.DB, cfg *config.Config) *gin.Engine {
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), corsMiddleware(cfg))

	repos := repo.New(db)
	aftersales := service.NewAfterSalesClient(cfg.AfterSales.BaseURL, cfg.AfterSales.InternalToken)
	svc := service.NewAgentService(repos, aftersales)
	adminH := admin.NewHandlers(svc)
	agentH := agentapi.NewHandler(svc)
	internalH := internalapi.NewHandler(svc, cfg.Auth.InternalToken)

	scheduler.NewJobRetentionScheduler(svc, cfg.Jobs.RetentionDays).Start()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "agentscenter"})
	})

	v1 := r.Group("/api/v1")
	jwtMgr := jwtmgr.NewManager(cfg.Auth.JWTSecret)

	agentGroup := v1.Group("/agent")
	{
		agentGroup.POST("/register", agentH.Register)
		authed := agentGroup.Group("")
		authed.Use(agentH.AuthRequired())
		{
			authed.POST("/heartbeat", agentH.Heartbeat)
			authed.GET("/jobs/claim", agentH.ClaimJobs)
			authed.POST("/jobs/:id/report", agentH.ReportJob)
		}
	}

	adminGroup := v1.Group("/admin")
	adminGroup.Use(adminmw.AdminAuth(&cfg.Auth, jwtMgr))
	admin.RegisterRoutes(adminGroup, adminH)

	internalGroup := v1.Group("/internal")
	internalGroup.Use(internalH.AuthRequired())
	internalGroup.POST("/jobs", internalH.CreateJob)
	internalGroup.GET("/shops", internalH.ListShops)
	internalGroup.POST("/assignments", internalH.UpsertAssignment)
	internalGroup.POST("/assignments/trigger", internalH.TriggerByShop)

	return r
}

func corsMiddleware(cfg *config.Config) gin.HandlerFunc {
	origins := cfg.CORS.AllowOrigins
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		allowed := origin == ""
		for _, o := range origins {
			if o == origin || o == "*" {
				allowed = true
				break
			}
		}
		if allowed && origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
		}
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
        c.Header("Access-Control-Allow-Headers", "Content-Type,Authorization,X-Agent-Key,X-Agent-Secret,X-Internal-Token")
		c.Header("Access-Control-Allow-Credentials", "true")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
