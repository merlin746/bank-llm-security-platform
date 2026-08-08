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
 }
 
 func NewHandler(
 	cacheClient *cache.Cache,
 	accessCtrlClient *contract.AccessControlClient,
 	complianceClient *contract.CompliancePolicyClient,
 	reconClient *contract.NodeReconciliationClient,
 	mqClient *mq.MQClient,
 ) *Handler {
 	return &Handler{
 		cacheClient:      cacheClient,
 		accessCtrlClient: accessCtrlClient,
 		complianceClient: complianceClient,
 		reconClient:      reconClient,
 		mqClient:         mqClient,
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
 	}
 }
