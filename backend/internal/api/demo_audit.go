package api

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/chainwise/backend/internal/model"
)

type demoRequest struct {
	ID             string        `json:"id"`
	BusinessType   string        `json:"businessType,omitempty"`
	Status         string        `json:"status"`
	Result         string        `json:"result,omitempty"`
	RiskTip        string        `json:"riskTip,omitempty"`
	RequestedLevel string        `json:"requestedLevel,omitempty"`
	RequestID      string        `json:"requestId"`
	Owner          string        `json:"owner"`
	Department     string        `json:"department"`
	Kind           string        `json:"kind"`
	Time           string        `json:"time"`
	Prompt         string        `json:"prompt,omitempty"`
	Role           string        `json:"role,omitempty"`
	DataLevel      string        `json:"dataLevel,omitempty"`
	Action         string        `json:"action,omitempty"`
	Verdict        string        `json:"verdict"`
	TotalLatencyMs int           `json:"totalLatencyMs"`
	Stages         []demoStage   `json:"stages"`
	Detection      demoDetection `json:"detection"`
	AlertIDs       []int         `json:"alertIds"`
}

type demoAlert struct {
	ID             int           `json:"id"`
	Owner          string        `json:"owner"`
	Department     string        `json:"department"`
	Time           string        `json:"time"`
	Node           string        `json:"node"`
	Type           string        `json:"type"`
	DataLevel      string        `json:"dataLevel"`
	Message        string        `json:"message"`
	Severity       string        `json:"severity"`
	Status         string        `json:"status"`
	RequestID      string        `json:"requestId"`
	Evidence       string        `json:"evidence"`
	Recommendation string        `json:"recommendation"`
	Detection      demoDetection `json:"detection"`
	Review         *riskReview   `json:"review,omitempty"`
}

type riskReview struct {
	Decision string `json:"decision"`
	Note     string `json:"note"`
	Reviewer string `json:"reviewer"`
	Time     string `json:"time"`
}

// The bridge stores demo traces in memory, matching its existing user store.
// Requests and their alerts are inserted and evicted together under one lock.
type demoAuditStore struct {
	mu           sync.RWMutex
	requests     map[string]demoRequest
	requestOrder []string
	alerts       map[int]demoAlert
	alertOrder   []int
	nextAlertID  int
}

const maxDemoRequests = 500

var demoRequestSequence atomic.Uint64

func newDemoRequestID() string {
	return fmt.Sprintf("req-%d-%d", time.Now().UnixNano(), demoRequestSequence.Add(1))
}

func demoTimestamp(t time.Time) string {
	return t.In(time.FixedZone("Asia/Shanghai", 8*60*60)).Format("2006-01-02 15:04:05")
}

func newDemoAuditStore() *demoAuditStore {
	s := &demoAuditStore{requests: map[string]demoRequest{}, alerts: map[int]demoAlert{}, nextAlertID: 4}
	detection := demoDetection{Source: "mock", Mode: "mock", Reason: "历史演示告警，未保存原始请求记录，无法关联请求链路"}
	seeds := []demoAlert{
		{ID: 1, Time: "2026-08-29 14:32:11", Node: "RAG 检索节点", Type: "hash-mismatch", Message: "节点 Hash 与链上不一致", Severity: "high", Status: "open",
			Evidence: "历史演示数据：节点 Hash 与链上记录不一致；未保存原始请求记录，无法关联请求链路。", Recommendation: "核对原始数据与链上记录，复核节点写入权限。"},
		{ID: 2, Time: "2026-08-29 13:05:44", Node: "访问节点", Type: "privilege", Message: "越权访问密级 L4 数据被拦截", Severity: "high", Status: "blocked",
			Evidence: "历史演示数据：越权访问 L4 被拦截；未保存原始请求记录，无法关联请求链路。", Recommendation: "核实用户职责与数据授权，按审批流程申请权限后重新发起请求。"},
		{ID: 3, Time: "2026-08-29 11:20:03", Node: "推理节点", Type: "jailbreak", Message: "检测到 Prompt 越狱意图", Severity: "high", Status: "blocked",
			Evidence: "历史演示数据：检测到越狱意图；未保存原始 Prompt 与请求记录，无法关联请求链路。", Recommendation: "核查请求来源与输入内容，保留拦截记录。"},
	}
	for _, alert := range seeds {
		alert.Detection = detection
		alert.Department = "零售业务部"
		alert.DataLevel = "L2"
		if alert.ID == 3 {
			alert.DataLevel = "L1"
		}
		s.alerts[alert.ID] = alert
	}
	s.alertOrder = []int{3, 2, 1}
	return s
}

func (s *demoAuditStore) record(request *demoRequest, alert *demoAlert) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if alert != nil {
		alert.ID = s.nextAlertID
		s.nextAlertID++
		alert.RequestID, alert.Time, alert.Detection = request.RequestID, request.Time, request.Detection
		alert.Owner, alert.Department, alert.DataLevel = request.Owner, request.Department, request.DataLevel
		request.AlertIDs = []int{alert.ID}
		s.alerts[alert.ID] = *alert
		s.alertOrder = append(s.alertOrder, alert.ID)
	}
	request.ID = request.RequestID
	s.requests[request.RequestID] = *request
	s.requestOrder = append(s.requestOrder, request.RequestID)
	if len(s.requestOrder) > maxDemoRequests {
		oldest := s.requestOrder[0]
		for _, alertID := range s.requests[oldest].AlertIDs {
			delete(s.alerts, alertID)
		}
		delete(s.requests, oldest)
		s.requestOrder = s.requestOrder[1:]
		retained := s.alertOrder[:0]
		for _, alertID := range s.alertOrder {
			if _, exists := s.alerts[alertID]; exists {
				retained = append(retained, alertID)
			}
		}
		s.alertOrder = retained
	}
}

func (s *demoAuditStore) listAlerts() []demoAlert {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]demoAlert, 0, len(s.alertOrder))
	for i := len(s.alertOrder) - 1; i >= 0; i-- {
		items = append(items, s.alerts[s.alertOrder[i]])
	}
	return items
}

func (h *Handler) DemoAuditAlertDetail(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	h.demoAudit.mu.RLock()
	alert, exists := h.demoAudit.alerts[id]
	h.demoAudit.mu.RUnlock()
	if err != nil || !exists {
		c.JSON(http.StatusNotFound, model.Error(404, "告警不存在或记录已过期"))
		return
	}
	if !canReadAlert(currentUser(c), alert) {
		c.JSON(http.StatusForbidden, model.Error(403, "该告警不在当前账号的授权范围内"))
		return
	}
	c.JSON(http.StatusOK, model.Success(alert))
}

func (h *Handler) DemoAuditRequestDetail(c *gin.Context) {
	h.demoAudit.mu.RLock()
	request, exists := h.demoAudit.requests[c.Param("requestId")]
	h.demoAudit.mu.RUnlock()
	if !exists {
		c.JSON(http.StatusNotFound, model.Error(404, "请求不存在或记录已过期"))
		return
	}
	if !canReadRequest(currentUser(c), request) {
		c.JSON(http.StatusForbidden, model.Error(403, "该请求不在当前账号的授权范围内"))
		return
	}
	c.JSON(http.StatusOK, model.Success(request))
}

func (h *Handler) DemoWorkspaceRequests(c *gin.Context) {
	c.JSON(http.StatusOK, model.Success(h.demoAudit.visibleRequests(currentUser(c), false)))
}
