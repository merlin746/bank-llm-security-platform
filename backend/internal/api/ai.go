package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/chainwise/backend/internal/model"
)

const aiProxyTimeout = 15 * time.Second

// proxyToAI 将请求转发至 FastAPI AI 服务并回传响应
func (h *Handler) proxyToAI(c *gin.Context, path string) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.Error(400, "failed to read request body"))
		return
	}

	target := h.aiBaseURL + path
	req, err := http.NewRequestWithContext(
		c.Request.Context(), http.MethodPost, target, bytes.NewReader(body),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Error(500, "failed to build AI request"))
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: aiProxyTimeout}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, model.Error(502, "AI service unavailable: "+err.Error()))
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusBadGateway, model.Error(502, "failed to read AI response"))
		return
	}

	// 透传 AI 服务的 JSON 响应
	var payload interface{}
	if json.Unmarshal(respBody, &payload) == nil {
		c.JSON(resp.StatusCode, payload)
		return
	}
	c.Data(resp.StatusCode, "application/json", respBody)
}

// AIDetectPrompt 转发 Prompt 攻击检测
func (h *Handler) AIDetectPrompt(c *gin.Context) {
	h.proxyToAI(c, "/api/prompt/detect")
}

// AIDesensitize 转发输出脱敏与合规检测
func (h *Handler) AIDesensitize(c *gin.Context) {
	h.proxyToAI(c, "/api/output/desensitize")
}

// AIRiskScore 转发用户行为风险评分
func (h *Handler) AIRiskScore(c *gin.Context) {
	h.proxyToAI(c, "/api/risk/score")
}
