package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/chainwise/backend/internal/model"
)

const aiCallTimeout = 15 * time.Second

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

	client := &http.Client{Timeout: aiCallTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var out map[string]interface{}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func strOr(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// DemoLogin 登录（演示：任意账号密码签发 token）
func (h *Handler) DemoLogin(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
	}
	_ = c.ShouldBindJSON(&req)
	if req.Username == "" {
		req.Username = "demo"
	}
	c.JSON(http.StatusOK, model.Success(gin.H{
		"token": fmt.Sprintf("demo-token-%d", time.Now().UnixNano()),
		"user": gin.H{
			"username":  req.Username,
			"role":      "风控审核员",
			"dataLevel": "L3",
		},
	}))
}

// DemoAttackTest Prompt 注入攻防测试：真实调用 FastAPI 输入检测
func (h *Handler) DemoAttackTest(c *gin.Context) {
	var req struct {
		Prompt string `json:"prompt"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Error(400, "invalid request"))
		return
	}

	start := time.Now()
	stages := []demoStage{
		{Key: "auth", Name: "身份认证", Owner: "B", Status: "pass", LatencyMs: 1, Message: "Token 校验通过"},
	}

	blocked := false
	message := "未检测到注入/越狱意图"
	extra := map[string]interface{}{"riskScore": 12, "riskType": "normal"}
	if result, err := h.callAI("/api/prompt/detect", map[string]interface{}{"text": req.Prompt}); err == nil {
		if isAttack, ok := result["is_attack"].(bool); ok && isAttack {
			blocked = true
			message = "检测到越狱/注入意图：" + strOr(result["reason"])
			extra = map[string]interface{}{
				"confidence": result["confidence"],
				"riskScore":  90,
				"riskType":   "jailbreak",
			}
		} else if conf, ok := result["confidence"].(float64); ok {
			extra["confidence"] = conf
		}
	} else {
		message = "AI 服务未就绪，按规则层判定"
	}

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
		Status: "pass", LatencyMs: 2, Message: "角色/密级/频次校验通过（Redis 缓存）",
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
	c.JSON(http.StatusOK, model.Success(gin.H{
		"requestId":      fmt.Sprintf("req-%d", time.Now().UnixNano()%100000),
		"prompt":         req.Prompt,
		"verdict":        verdict,
		"totalLatencyMs": total,
		"stages":         stages,
	}))
}

// DemoAccessTest 越权访问攻防测试
func (h *Handler) DemoAccessTest(c *gin.Context) {
	var req struct {
		Role      string `json:"role"`
		DataLevel string `json:"dataLevel"`
		Action    string `json:"action"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Error(400, "invalid request"))
		return
	}

	roleLevel := map[string]string{"普通柜员": "L2", "客服坐席": "L2", "风控审核员": "L3", "管理员": "L4"}
	allowed := roleLevel[req.Role]
	if allowed == "" {
		allowed = "L1"
	}
	canAccess := allowed >= req.DataLevel

	stages := []demoStage{
		{Key: "auth", Name: "身份认证", Owner: "B", Status: "pass", LatencyMs: 1, Message: "Token 校验通过"},
		{Key: "input-risk", Name: "AI 输入风险检测", Owner: "C", Status: "pass", LatencyMs: 8, Message: "未检测到注入/越狱意图"},
	}
	accessStatus := "pass"
	accessMsg := "角色/密级/频次校验通过（Redis 缓存）"
	accessExtra := map[string]interface{}{"requestedLevel": req.DataLevel, "allowedLevel": allowed}
	if !canAccess {
		accessStatus = "block"
		accessMsg = fmt.Sprintf("角色 [%s] 无权访问密级 [%s] 数据", req.Role, req.DataLevel)
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
	c.JSON(http.StatusOK, model.Success(gin.H{
		"requestId":      fmt.Sprintf("req-%d", time.Now().UnixNano()%100000),
		"role":           req.Role,
		"dataLevel":      req.DataLevel,
		"action":         req.Action,
		"verdict":        verdict,
		"totalLatencyMs": total,
		"stages":         stages,
	}))
}

// DemoStatsOverview 安全态势总览 KPI
func (h *Handler) DemoStatsOverview(c *gin.Context) {
	c.JSON(http.StatusOK, model.Success(gin.H{
		"totalRequests": 12894,
		"blockedToday":  321,
		"blockRate":     2.49,
		"highRiskUsers": 7,
	}))
}

// DemoStatsTrend 拦截量趋势（演示序列）
func (h *Handler) DemoStatsTrend(c *gin.Context) {
	hours := []string{"00:00", "04:00", "08:00", "12:00", "16:00", "20:00", "24:00"}
	blocked := []int{40, 22, 68, 95, 71, 52, 31}
	passed := []int{800, 620, 1500, 2300, 1900, 1400, 900}
	items := make([]gin.H, 0, len(hours))
	for i := range hours {
		items = append(items, gin.H{"time": hours[i], "blocked": blocked[i], "passed": passed[i]})
	}
	c.JSON(http.StatusOK, model.Success(items))
}

// DemoStatsRiskDistribution 风险类型分布（演示）
func (h *Handler) DemoStatsRiskDistribution(c *gin.Context) {
	c.JSON(http.StatusOK, model.Success([]gin.H{
		{"type": "Prompt 注入", "value": 128},
		{"type": "越权访问", "value": 96},
		{"type": "敏感数据泄露", "value": 54},
		{"type": "违规承诺", "value": 28},
		{"type": "频次异常", "value": 15},
	}))
}

// DemoStatsHighRiskUsers 高风险用户榜单（演示）
func (h *Handler) DemoStatsHighRiskUsers(c *gin.Context) {
	c.JSON(http.StatusOK, model.Success([]gin.H{
		{"name": "u_200731", "role": "普通柜员", "riskScore": 96, "lastAction": "越权查询密级 L4", "level": "高"},
		{"name": "u_100288", "role": "风控审核员", "riskScore": 87, "lastAction": "高频 Prompt 注入尝试", "level": "高"},
		{"name": "u_301445", "role": "客服坐席", "riskScore": 74, "lastAction": "导出敏感字段", "level": "中"},
		{"name": "u_400912", "role": "普通柜员", "riskScore": 68, "lastAction": "异常时段访问", "level": "中"},
	}))
}

// DemoAuditTopology 审计溯源四节点拓扑
func (h *Handler) DemoAuditTopology(c *gin.Context) {
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
	c.JSON(http.StatusOK, model.Success([]gin.H{
		{"id": 1, "time": "2026-08-29 14:32:11", "node": "RAG 检索节点", "type": "hash-mismatch", "message": "节点 Hash 与链上不一致"},
		{"id": 2, "time": "2026-08-29 13:05:44", "node": "访问节点", "type": "privilege", "message": "越权访问密级 L4 数据被拦截"},
		{"id": 3, "time": "2026-08-29 11:20:03", "node": "推理节点", "type": "jailbreak", "message": "检测到 Prompt 越狱意图"},
	}))
}
