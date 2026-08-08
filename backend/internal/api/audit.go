 package api
 
 import (
 	"net/http"
 	"strconv"
 
 	"github.com/gin-gonic/gin"
 
 	"github.com/chainwise/backend/internal/model"
 )
 
 // GetAnomalyStats 获取异常统计信息（对账拓扑图数据源）
 func (h *Handler) GetAnomalyStats(c *gin.Context) {
 	stats, err := h.reconClient.GetAnomalyStats()
 	if err != nil {
 		c.JSON(http.StatusInternalServerError, model.Error(500, "failed to get anomaly stats: "+err.Error()))
 		return
 	}
 	c.JSON(http.StatusOK, model.Success(stats))
 }
 
 // GetRequestList 分页查询请求列表（审计溯源列表）
 func (h *Handler) GetRequestList(c *gin.Context) {
 	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
 	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
 
 	ids, total, err := h.reconClient.GetRequestIDs(offset, limit)
 	if err != nil {
 		c.JSON(http.StatusInternalServerError, model.Error(500, "failed to get request list"))
 		return
 	}
 
 	// 获取每个请求的详细信息
 	items := make([]*model.ReconciliationResult, 0, len(ids))
 	for _, id := range ids {
 		result, err := h.reconClient.GetReconciliationResult(id)
 		if err != nil {
 			continue
 		}
 		items = append(items, result)
 	}
 
 	c.JSON(http.StatusOK, model.Success(model.PageResponse{
 		Items: items,
 		Total: total,
 		Page:  offset/limit + 1,
 		Size:  limit,
 	}))
 }
 
 // GetRequestDetail 获取单个请求的对账详情（溯源拓扑图数据源）
 func (h *Handler) GetRequestDetail(c *gin.Context) {
 	requestID := c.Param("requestId")
 
 	result, err := h.reconClient.GetReconciliationResult(requestID)
 	if err != nil {
 		c.JSON(http.StatusNotFound, model.Error(404, "request not found"))
 		return
 	}
 
 	// 同时返回原始节点 Hash 记录
 	// record, _ := h.reconClient.GetRequestRecord(requestID)
 
 	c.JSON(http.StatusOK, model.Success(gin.H{
 		"reconciliation": result,
 		// "record": record,
 	}))
 }
 
 // GetRecentAnomalies 获取近期异常列表（告警日志）
 func (h *Handler) GetRecentAnomalies(c *gin.Context) {
 	count, _ := strconv.Atoi(c.DefaultQuery("count", "20"))
 
 	// 优先从 Redis 缓存获取
 	if h.cacheClient != nil {
 		cached, err := h.cacheClient.GetRecentAnomalies(c.Request.Context(), count)
 		if err == nil && len(cached) > 0 {
 			c.JSON(http.StatusOK, model.Success(cached))
 			return
 		}
 	}
 
 	// 缓存未命中则从链上查询
 	ids, nodeTypes, err := h.reconClient.GetRecentAnomalies(count)
 	if err != nil {
 		c.JSON(http.StatusInternalServerError, model.Error(500, "failed to get anomalies"))
 		return
 	}
 
 	type AnomalyItem struct {
 		RequestID string          `json:"request_id"`
 		NodeType  string          `json:"node_type"`
 	}
 
 	items := make([]AnomalyItem, 0, len(ids))
 	for i := range ids {
 		nodeName := "UNKNOWN"
 		if i < len(nodeTypes) {
 			nodeName = nodeTypes[i].String()
 		}
 		items = append(items, AnomalyItem{
 			RequestID: ids[i],
 			NodeType:  nodeName,
 		})
 	}
 
 	c.JSON(http.StatusOK, model.Success(items))
 }
 
 // GetUserPermission 查询用户权限
 func (h *Handler) GetUserPermission(c *gin.Context) {
 	address := c.Param("address")
 
 	// 优先查缓存
 	if h.cacheClient != nil {
 		var cached model.UserPermission
 		if err := h.cacheClient.GetUserPermission(c.Request.Context(), address, &cached); err == nil {
 			c.JSON(http.StatusOK, model.Success(cached))
 			return
 		}
 	}
 
 	perm, err := h.accessCtrlClient.GetUserInfo(address)
 	if err != nil {
 		c.JSON(http.StatusNotFound, model.Error(404, "user not found"))
 		return
 	}
 
 	c.JSON(http.StatusOK, model.Success(perm))
 }
 
 // CheckAccess 校验访问权限
 func (h *Handler) CheckAccess(c *gin.Context) {
 	var req struct {
 		UserAddr  string `json:"user_addr" binding:"required"`
 		DataLevel int    `json:"data_level" binding:"required"`
 	}
 	if err := c.ShouldBindJSON(&req); err != nil {
 		c.JSON(http.StatusBadRequest, model.Error(400, "invalid request"))
 		return
 	}
 
 	allowed, reason, err := h.accessCtrlClient.CheckAccess(req.UserAddr, req.DataLevel)
 	if err != nil {
 		c.JSON(http.StatusInternalServerError, model.Error(500, "check access failed"))
 		return
 	}
 
 	c.JSON(http.StatusOK, model.Success(gin.H{
 		"allowed": allowed,
 		"reason":  reason,
 	}))
 }
 
 // GetActivePolicy 获取当前生效策略
 func (h *Handler) GetActivePolicy(c *gin.Context) {
 	version, err := h.complianceClient.GetActiveVersion()
 	if err != nil {
 		c.JSON(http.StatusInternalServerError, model.Error(500, "failed to get active policy"))
 		return
 	}
 	c.JSON(http.StatusOK, model.Success(version))
 }
 
 // GetActiveRules 获取当前生效规则列表
 func (h *Handler) GetActiveRules(c *gin.Context) {
 	rules, err := h.complianceClient.GetActiveRuleIDs()
 	if err != nil {
 		c.JSON(http.StatusInternalServerError, model.Error(500, "failed to get active rules"))
 		return
 	}
 	c.JSON(http.StatusOK, model.Success(rules))
 }
