package api

import (
	"github.com/gin-gonic/gin"

	"github.com/chainwise/backend/internal/cache"
	"github.com/chainwise/backend/internal/contract"
	"github.com/chainwise/backend/internal/mq"
)

type Handler struct {
	cacheClient      *cache.Cache
	accessCtrlClient *contract.AccessControlClient
	complianceClient *contract.CompliancePolicyClient
	reconClient      *contract.NodeReconciliationClient
	mqClient         *mq.MQClient
	aiBaseURL        string
	demoAudit        *demoAuditStore
	sessions         *sessionStore
	settings         *demoSettingsStore
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
		demoAudit:        newDemoAuditStore(),
		sessions:         newSessionStore(),
		settings:         newDemoSettingsStore(),
	}
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	r.Use(h.protectAPI)
	r.POST("/api/auth/login", h.DemoLogin)
	r.GET("/api/auth/me", h.DemoCurrentUser)
	r.POST("/api/auth/logout", h.DemoLogout)
	api := r.Group("/api/v1")
	api.GET("/health", h.Health)
	// Contract stubs have no verified customer scope, so they fail closed.
	api.GET("/audit/stats", denyUnscopedChainData)
	api.GET("/audit/requests", denyUnscopedChainData)
	api.GET("/audit/requests/:requestId", denyUnscopedChainData)
	api.GET("/audit/anomalies", denyUnscopedChainData)
	api.GET("/permission/users/:address", denyUnscopedChainData)
	api.POST("/permission/check-access", denyUnscopedChainData)
	api.GET("/policy/active", denyUnscopedChainData)
	api.GET("/policy/rules", denyUnscopedChainData)
	ai := api.Group("/ai", h.requireGatewayEnabled)
	ai.POST("/prompt/detect", h.AIDetectPrompt)
	ai.POST("/output/desensitize", h.AIDesensitize)
	ai.POST("/risk/score", h.AIRiskScore)

	bridge := r.Group("/api")
	gateway := bridge.Group("/gateway", h.requireGatewayEnabled)
	gateway.POST("/attack-test", h.DemoAttackTest)
	gateway.POST("/access-test", h.DemoAccessTest)
	bridge.GET("/business/requests", h.DemoBusinessRequests)
	bridge.POST("/business/requests", h.DemoBusinessSubmit)
	bridge.GET("/business/requests/:requestId", h.DemoBusinessRequestDetail)
	bridge.GET("/workspace/requests", h.DemoWorkspaceRequests)
	bridge.GET("/stats/overview", h.DemoStatsOverview)
	bridge.GET("/stats/trend", h.DemoStatsTrend)
	bridge.GET("/stats/risk-distribution", h.DemoStatsRiskDistribution)
	bridge.GET("/stats/high-risk-users", h.DemoStatsHighRiskUsers)
	bridge.GET("/audit/topology", h.DemoAuditTopology)
	bridge.GET("/audit/alerts", h.DemoAuditAlerts)
	bridge.GET("/audit/alerts/:id", h.DemoAuditAlertDetail)
	bridge.GET("/audit/requests", h.DemoAuditRequests)
	bridge.GET("/audit/requests/:requestId", h.DemoAuditRequestDetail)
	bridge.POST("/risk/reviews", h.DemoRiskReview)
	bridge.GET("/users", h.DemoListUsers)
	bridge.POST("/users", h.DemoCreateUser)
	bridge.GET("/users/:id", h.DemoGetUser)
	bridge.PUT("/users/:id", h.DemoUpdateUser)
	bridge.DELETE("/users/:id", h.DemoDeleteUser)
	bridge.GET("/roles", h.DemoListRoles)
	bridge.GET("/data-levels", h.DemoListDataLevels)
	bridge.GET("/system/settings", h.DemoGetSettings)
	bridge.PUT("/system/settings", h.DemoUpdateSettings)
}
