 package api
 
 import (
 	"github.com/gin-gonic/gin"
 
 	"github.com/chainwise/backend/internal/cache"
 	"github.com/chainwise/backend/internal/contract"
 	"github.com/chainwise/backend/internal/mq"
 )
 
 type Handler struct {
 	cacheClient          *cache.Cache
 	accessCtrlClient     *contract.AccessControlClient
 	complianceClient     *contract.CompliancePolicyClient
	reconClient          *contract.NodeReconciliationClient
	mqClient             *mq.MQClient
	aiBaseURL            string
}

func NewHandler(
	cacheClient *cache.Cache,
	accessCtrlClient *contract.AccessControlClient,
	complianceClient *contract.CompliancePolicyClient,
	reconClient *contract.NodeReconciliationClient,
	mqClient *mq.MQClient,
	aiBaseURL string,
) *Handler {
	return &Handler{
		cacheClient:      cacheClient,
		accessCtrlClient: accessCtrlClient,
		complianceClient: complianceClient,
		reconClient:      reconClient,
		mqClient:         mqClient,
		aiBaseURL:        aiBaseURL,
	}
}
 
 func (h *Handler) RegisterRoutes(r *gin.Engine) {
 	api := r.Group("/api/v1")
 	{
 		// 健康检查
 		api.GET("/health", h.Health)
 
 		// 审计溯源
 		audit := api.Group("/audit")
 		{
 			audit.GET("/stats", h.GetAnomalyStats)
 			audit.GET("/requests", h.GetRequestList)
 			audit.GET("/requests/:requestId", h.GetRequestDetail)
 			audit.GET("/anomalies", h.GetRecentAnomalies)
 		}
 
 		// 权限管理
 		perm := api.Group("/permission")
 		{
 			perm.GET("/users/:address", h.GetUserPermission)
 			perm.POST("/check-access", h.CheckAccess)
 		}
 
 		// 策略管理
		policy := api.Group("/policy")
		{
			policy.GET("/active", h.GetActivePolicy)
			policy.GET("/rules", h.GetActiveRules)
		}

		// AI 服务代理（转发 FastAPI 微服务）
		ai := api.Group("/ai")
		{
			ai.POST("/prompt/detect", h.AIDetectPrompt)
			ai.POST("/output/desensitize", h.AIDesensitize)
			ai.POST("/risk/score", h.AIRiskScore)
		}
	}

	// 成员2前端联调路由（vite proxy 传入 /api/... 前缀）
	bridge := r.Group("/api")
	{
		bridge.POST("/auth/login", h.DemoLogin)
		bridge.POST("/gateway/attack-test", h.DemoAttackTest)
		bridge.POST("/gateway/access-test", h.DemoAccessTest)
		bridge.GET("/stats/overview", h.DemoStatsOverview)
		bridge.GET("/stats/trend", h.DemoStatsTrend)
		bridge.GET("/stats/risk-distribution", h.DemoStatsRiskDistribution)
		bridge.GET("/stats/high-risk-users", h.DemoStatsHighRiskUsers)
		bridge.GET("/audit/topology", h.DemoAuditTopology)
		bridge.GET("/audit/alerts", h.DemoAuditAlerts)
	}
}
