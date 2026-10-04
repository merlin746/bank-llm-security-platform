package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/chainwise/backend/internal/model"
)

const aiCallTimeout = 4 * time.Second

// demoStage 与成员2前端 PipelineStages 的 stages[] 结构保持一致
type demoStage struct {
	Key       string                 `json:"key"`
	Name      string                 `json:"name"`
	Owner     string                 `json:"owner"`
	Status    string                 `json:"status"`
	LatencyMs int                    `json:"latencyMs"`
	Message   string                 `json:"message"`
	Extra     map[string]interface{} `json:"extra,omitempty"`
}

// callAI 调用 FastAPI AI 服务并解析 JSON
func (h *Handler) callAI(path string, payload map[string]interface{}) (map[string]interface{}, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, h.aiBaseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: time.Duration(h.settings.snapshot().Runtime.AITimeoutMs) * time.Millisecond}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("AI 服务返回异常状态（HTTP %d）", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var out map[string]interface{}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, errors.New("AI 服务返回无效的检测响应")
	}
	if _, ok := out["is_attack"].(bool); !ok {
		return nil, errors.New("AI 服务未返回有效的攻击判定")
	}
	return out, nil
}

func strOr(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// DemoLogin 校验演示账号凭据，返回账号当前的角色与密级。
func (h *Handler) DemoLogin(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Error(400, "请输入账号和密码"))
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || strings.TrimSpace(req.Password) == "" {
		c.JSON(http.StatusBadRequest, model.Error(400, "请输入账号和密码"))
		return
	}
	u, valid := demoUsers.authenticate(req.Username, req.Password)
	if !valid {
		c.JSON(http.StatusUnauthorized, model.Error(401, "账号或密码错误"))
		return
	}
	token, err := h.sessions.issue(u.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Error(500, "暂时无法创建登录状态"))
		return
	}
	c.JSON(http.StatusOK, model.Success(gin.H{
		"token": token,
		"user":  u,
	}))
}

// DemoAttackTest Prompt 注入攻防测试：真实调用 FastAPI 输入检测
func (h *Handler) DemoAttackTest(c *gin.Context) {
	user := currentUser(c)
	if !canReadData(user, user.Username, businessDepartment(user), user.DataLevel) {
		c.JSON(http.StatusForbidden, model.Error(403, "当前账号没有业务数据授权范围"))
		return
	}
	var req struct {
		Prompt string `json:"prompt" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Error(400, "invalid request"))
		return
	}
	req.Prompt = strings.TrimSpace(req.Prompt)
	if req.Prompt == "" {
		c.JSON(http.StatusBadRequest, model.Error(400, "prompt is required"))
		return
	}

	start := time.Now()
	stages := []demoStage{
		{Key: "auth", Name: "身份认证", Owner: "B", Status: "pass", LatencyMs: 1, Message: "Token 校验通过"},
	}

	blocked, message, extra, detection := h.detectDemoPrompt(req.Prompt)

	inputStatus := "pass"
	if blocked {
		inputStatus = "block"
	}
	stages = append(stages, demoStage{
		Key: "input-risk", Name: "AI 输入风险检测", Owner: "C",
		Status: inputStatus, LatencyMs: int(time.Since(start).Milliseconds()) + 1,
		Message: message, Extra: extra,
	})
	stages = append(stages, demoStage{
		Key: "access-control", Name: "权限/密级校验", Owner: "A",
		Status: "pass", LatencyMs: 2, Message: "本地演示账号、数据密级与授权部门校验通过",
	})

	if blocked {
		stages = append(stages,
			demoStage{Key: "infer", Name: "LLM 推理节点", Owner: "-", Status: "skip", Message: "已被拦截，未进入推理"},
			demoStage{Key: "output-sanitize", Name: "输出脱敏与合规", Owner: "C", Status: "skip", Message: "-"},
		)
	} else {
		stages = append(stages,
			demoStage{Key: "infer", Name: "LLM 推理节点", Owner: "-", Status: "pass", LatencyMs: 46, Message: "推理完成"},
			demoStage{Key: "output-sanitize", Name: "输出脱敏与合规", Owner: "C", Status: "pass", LatencyMs: 9, Message: "敏感信息已掩码，合规评分 98"},
		)
	}

	verdict := "pass"
	if blocked {
		verdict = "block"
	}
	total := 0
	for _, s := range stages {
		total += s.LatencyMs
	}
	result := demoRequest{
		RequestID: newDemoRequestID(), Kind: "attack", Time: demoTimestamp(start),
		Owner: user.Username, Department: businessDepartment(user), Role: user.Role, DataLevel: user.DataLevel,
		Prompt: req.Prompt, Verdict: verdict, TotalLatencyMs: total,
		Stages: stages, Detection: detection, AlertIDs: []int{},
	}
	var alert *demoAlert
	if blocked {
		alert = &demoAlert{Node: "访问节点", Type: "jailbreak", Message: message,
			Severity: "high", Status: "blocked", Evidence: fmt.Sprintf("输入：%s；检测依据：%s；风险评分：%v/100", req.Prompt, message, extra["riskScore"]),
			Recommendation: "核查请求来源与输入内容，保留拦截记录；确认安全后再发起新请求。"}
	}
	h.demoAudit.record(&result, alert)
	c.JSON(http.StatusOK, model.Success(result))
}

// DemoAccessTest 越权访问攻防测试
func (h *Handler) DemoAccessTest(c *gin.Context) {
	user := currentUser(c)
	if !canReadData(user, user.Username, businessDepartment(user), user.DataLevel) {
		c.JSON(http.StatusForbidden, model.Error(403, "当前账号没有业务数据授权范围"))
		return
	}
	var req struct {
		Role      string `json:"role"`
		DataLevel string `json:"dataLevel" binding:"required,oneof=L1 L2 L3 L4"`
		Action    string `json:"action"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Error(400, "invalid request"))
		return
	}

	allowedRank := levelRank(user.DataLevel)
	if req.Role != "" && req.Role != user.Role {
		c.JSON(http.StatusForbidden, model.Error(403, "请求角色必须与当前登录账号一致"))
		return
	}
	allowed := fmt.Sprintf("L%d", allowedRank)
	canAccess := allowedRank >= levelRank(req.DataLevel)

	detection := demoDetection{Source: "gateway-rules", Mode: "rules", Reason: "本次执行角色与密级规则校验，未执行 Prompt 检测"}
	stages := []demoStage{
		{Key: "auth", Name: "身份认证", Owner: "B", Status: "pass", LatencyMs: 1, Message: "Token 校验通过"},
		{Key: "input-risk", Name: "AI 输入风险检测", Owner: "C", Status: "skip", Message: "本次为权限测试，未提交 Prompt"},
	}
	accessStatus := "pass"
	accessMsg := "当前账号的客户密级校验通过（本地演示授权）"
	accessExtra := map[string]interface{}{"requestedLevel": req.DataLevel, "allowedLevel": allowed, "detection": detection, "simulatedRole": req.Role}
	if !canAccess {
		accessStatus = "block"
		accessMsg = fmt.Sprintf("账号 [%s] 的有效客户密级 [%s] 不允许访问 [%s] 数据", user.Username, allowed, req.DataLevel)
	}
	stages = append(stages, demoStage{
		Key: "access-control", Name: "权限/密级校验", Owner: "A",
		Status: accessStatus, LatencyMs: 2, Message: accessMsg, Extra: accessExtra,
	})
	if canAccess {
		stages = append(stages,
			demoStage{Key: "infer", Name: "LLM 推理节点", Owner: "-", Status: "pass", LatencyMs: 46, Message: "推理完成"},
			demoStage{Key: "output-sanitize", Name: "输出脱敏与合规", Owner: "C", Status: "pass", LatencyMs: 9, Message: "敏感信息已掩码，合规评分 98"},
		)
	} else {
		stages = append(stages,
			demoStage{Key: "infer", Name: "LLM 推理节点", Owner: "-", Status: "skip", Message: "已被拦截，未进入推理"},
			demoStage{Key: "output-sanitize", Name: "输出脱敏与合规", Owner: "C", Status: "skip", Message: "-"},
		)
	}

	verdict := "pass"
	if !canAccess {
		verdict = "block"
	}
	total := 0
	for _, s := range stages {
		total += s.LatencyMs
	}
	result := demoRequest{
		RequestID: newDemoRequestID(), Kind: "access", Time: demoTimestamp(time.Now()),
		Owner: user.Username, Department: businessDepartment(user), Role: user.Role, DataLevel: user.DataLevel, RequestedLevel: req.DataLevel, Action: req.Action,
		Verdict: verdict, TotalLatencyMs: total, Stages: stages, Detection: detection, AlertIDs: []int{},
	}
	var alert *demoAlert
	if !canAccess {
		alert = &demoAlert{Node: "访问节点", Type: "privilege", Message: accessMsg,
			Severity: "high", Status: "blocked",
			Evidence:       fmt.Sprintf("账号：%s；职责：%s；有效客户密级：%s；请求密级：%s；操作：%s", user.Username, user.Role, allowed, req.DataLevel, req.Action),
			Recommendation: "核实用户职责与数据授权，按审批流程申请权限后重新发起请求。"}
	}
	h.demoAudit.record(&result, alert)
	c.JSON(http.StatusOK, model.Success(result))
}

// DemoAuditTopology 审计溯源四节点拓扑
func (h *Handler) DemoAuditTopology(c *gin.Context) {
	h.demoAudit.mu.RLock()
	evidence, exists := h.demoAudit.alerts[1]
	h.demoAudit.mu.RUnlock()
	if !exists || !canReadAlert(currentUser(c), evidence) {
		c.JSON(http.StatusOK, model.Success(gin.H{
			"nodes": []gin.H{}, "edges": []gin.H{}, "chainStatus": "unavailable",
			"alert": "当前账号的密级或业务范围未获此演示拓扑证据授权",
		}))
		return
	}
	c.JSON(http.StatusOK, model.Success(gin.H{
		"nodes": []gin.H{
			{"id": "access", "name": "访问节点", "hash": "0x3a9f21c4e8b7", "status": "normal", "x": 80, "y": 200},
			{"id": "rag", "name": "RAG 检索节点", "hash": "0x8c41de2a5f90", "status": "tampered", "x": 300, "y": 200},
			{"id": "inference", "name": "推理节点", "hash": "0x51b7e0d9c3a2", "status": "normal", "x": 520, "y": 200},
			{"id": "warehouse", "name": "数仓节点", "hash": "0x9e6d4f1b8a37", "status": "normal", "x": 740, "y": 200},
		},
		"edges": []gin.H{
			{"from": "access", "to": "rag"},
			{"from": "rag", "to": "inference"},
			{"from": "inference", "to": "warehouse"},
		},
		"chainStatus": "inconsistent",
		"alert":       "RAG 检索节点 Hash 与链上记录不一致，疑似被篡改",
	}))
}

// DemoAuditAlerts 告警日志列表
func (h *Handler) DemoAuditAlerts(c *gin.Context) {
	c.JSON(http.StatusOK, model.Success(h.demoAudit.visibleAlerts(currentUser(c))))
}
