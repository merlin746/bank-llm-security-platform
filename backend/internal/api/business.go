package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/chainwise/backend/internal/model"
	"github.com/gin-gonic/gin"
)

func (s *demoAuditStore) visibleRequests(u *demoUser, businessOnly bool) []demoRequest {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := []demoRequest{}
	for i := len(s.requestOrder) - 1; i >= 0; i-- {
		request := s.requests[s.requestOrder[i]]
		if (!businessOnly || request.Kind == "business") && canReadRequest(u, request) {
			items = append(items, request)
		}
	}
	return items
}

func (s *demoAuditStore) visibleAlerts(u *demoUser) []demoAlert {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := []demoAlert{}
	for i := len(s.alertOrder) - 1; i >= 0; i-- {
		alert := s.alerts[s.alertOrder[i]]
		if canReadAlert(u, alert) {
			items = append(items, alert)
		}
	}
	return items
}

func (h *Handler) DemoBusinessRequests(c *gin.Context) {
	c.JSON(http.StatusOK, model.Success(h.demoAudit.visibleRequests(currentUser(c), true)))
}

func (h *Handler) DemoAuditRequests(c *gin.Context) {
	c.JSON(http.StatusOK, model.Success(h.demoAudit.visibleRequests(currentUser(c), false)))
}

// Business results and risk notices remain attached to the authenticated owner.
// Client-supplied owner, role, scope and department fields are ignored.
func (h *Handler) DemoBusinessSubmit(c *gin.Context) {
	var req struct {
		Prompt       string `json:"prompt"`
		DataLevel    string `json:"dataLevel"`
		BusinessType string `json:"businessType"`
		Action       string `json:"action"`
	}
	if c.ShouldBindJSON(&req) != nil || strings.TrimSpace(req.Prompt) == "" {
		c.JSON(http.StatusBadRequest, model.Error(400, "请输入业务问题"))
		return
	}
	u := currentUser(c)
	if req.DataLevel == "" {
		req.DataLevel = "L1"
	}
	if !isValidDataLevel(req.DataLevel) {
		c.JSON(http.StatusBadRequest, model.Error(400, "无效的数据密级"))
		return
	}
	department := businessDepartment(u)
	if !canReadData(u, u.Username, department, req.DataLevel) {
		c.JSON(http.StatusForbidden, model.Error(403, "业务请求超出账号的数据密级授权"))
		return
	}
	if req.BusinessType == "" {
		req.BusinessType = "业务咨询"
	}
	blocked, message, extra, detection := h.detectDemoPrompt(strings.TrimSpace(req.Prompt))
	request := demoRequest{
		RequestID: newDemoRequestID(), Kind: "business", Time: demoTimestamp(time.Now()),
		Owner: u.Username, Department: department, Role: u.Role, DataLevel: req.DataLevel,
		Prompt: strings.TrimSpace(req.Prompt), BusinessType: req.BusinessType, Action: req.Action,
		Verdict: "pass", Status: "completed", Result: "已按授权范围完成业务咨询。请依据银行现行业务流程继续办理。",
		RiskTip: "当前结果仅用于本账号获授权的业务办理。", AlertIDs: []int{}, Detection: detection,
		Stages: []demoStage{{Key: "auth", Name: "身份认证", Status: "pass", Message: "账号认证通过"},
			{Key: "access-control", Name: "权限与数据范围", Status: "pass", Message: "已校验当前账号、密级及业务归属"}},
	}
	var alert *demoAlert
	if blocked {
		request.Verdict, request.Status = "block", "blocked"
		request.Result, request.RiskTip = "本次请求被安全策略拦截，未进入模型推理。", message
		alert = &demoAlert{Node: "访问节点", Type: "jailbreak", Message: message, Severity: "high", Status: "blocked",
			Evidence: request.Prompt + "；检测依据：" + message, Recommendation: "核查输入内容并按授权范围重新发起业务请求。"}
	}
	request.Stages = append(request.Stages, demoStage{Key: "input-risk", Name: "输入风险检测", Status: request.Verdict, Message: message, Extra: extra})
	h.demoAudit.record(&request, alert)
	c.JSON(http.StatusOK, model.Success(request))
}

func (h *Handler) DemoBusinessRequestDetail(c *gin.Context) {
	h.demoAudit.mu.RLock()
	request, exists := h.demoAudit.requests[c.Param("requestId")]
	h.demoAudit.mu.RUnlock()
	if !exists || request.Kind != "business" {
		c.JSON(http.StatusNotFound, model.Error(404, "业务请求不存在或记录已过期"))
		return
	}
	if !canReadRequest(currentUser(c), request) {
		c.JSON(http.StatusForbidden, model.Error(403, "该业务请求不在授权范围内"))
		return
	}
	c.JSON(http.StatusOK, model.Success(request))
}

func (h *Handler) DemoRiskReview(c *gin.Context) {
	var req struct {
		AlertID  int    `json:"alertId"`
		Decision string `json:"decision"`
		Note     string `json:"note"`
		Comment  string `json:"comment"`
	}
	if c.ShouldBindJSON(&req) != nil || req.AlertID <= 0 || (req.Decision != "confirmed" && req.Decision != "dismissed") {
		c.JSON(http.StatusBadRequest, model.Error(400, "请输入有效的告警与复核结论"))
		return
	}
	if req.Note == "" {
		req.Note = req.Comment
	}
	if strings.TrimSpace(req.Note) == "" || len([]rune(req.Note)) > 1000 {
		c.JSON(http.StatusBadRequest, model.Error(400, "请填写不超过 1000 字的复核意见"))
		return
	}
	h.demoAudit.mu.Lock()
	alert, exists := h.demoAudit.alerts[req.AlertID]
	if !exists {
		h.demoAudit.mu.Unlock()
		c.JSON(http.StatusNotFound, model.Error(404, "告警不存在或记录已过期"))
		return
	}
	if !canReadAlert(currentUser(c), alert) {
		h.demoAudit.mu.Unlock()
		c.JSON(http.StatusForbidden, model.Error(403, "该告警不在授权范围内"))
		return
	}
	if alert.Review != nil {
		h.demoAudit.mu.Unlock()
		c.JSON(http.StatusConflict, model.Error(409, "该告警已完成复核"))
		return
	}
	alert.Review = &riskReview{Decision: req.Decision, Note: strings.TrimSpace(req.Note), Reviewer: currentUser(c).Username, Time: demoTimestamp(time.Now())}
	alert.Status = req.Decision
	h.demoAudit.alerts[req.AlertID] = alert
	h.demoAudit.mu.Unlock()
	c.JSON(http.StatusOK, model.Success(alert))
}
