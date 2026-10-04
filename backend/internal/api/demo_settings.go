package api

import (
	"net/http"
	"sync"
	"time"

	"github.com/chainwise/backend/internal/model"
	"github.com/gin-gonic/gin"
)

type demoSettings struct {
	Policy struct {
		AllowGatewayTests bool `json:"allowGatewayTests"`
	} `json:"policy"`
	Runtime struct {
		AITimeoutMs int `json:"aiTimeoutMs"`
	} `json:"runtime"`
}

type demoSettingsStore struct {
	mu    sync.RWMutex
	value demoSettings
}

func newDemoSettingsStore() *demoSettingsStore {
	s := &demoSettingsStore{}
	s.value.Policy.AllowGatewayTests = true
	s.value.Runtime.AITimeoutMs = int(aiCallTimeout / time.Millisecond)
	return s
}

func (s *demoSettingsStore) snapshot() demoSettings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.value
}

func (h *Handler) requireGatewayEnabled(c *gin.Context) {
	if !h.settings.snapshot().Policy.AllowGatewayTests {
		c.AbortWithStatusJSON(http.StatusForbidden, model.Error(403, "系统策略已暂停业务测试"))
		return
	}
	c.Next()
}

func (h *Handler) DemoGetSettings(c *gin.Context) {
	c.JSON(http.StatusOK, model.Success(h.settings.snapshot()))
}

func (h *Handler) DemoUpdateSettings(c *gin.Context) {
	var req struct {
		Policy *struct {
			AllowGatewayTests *bool `json:"allowGatewayTests"`
		} `json:"policy"`
		Runtime *struct {
			AITimeoutMs *int `json:"aiTimeoutMs"`
		} `json:"runtime"`
	}
	if err := c.ShouldBindJSON(&req); err != nil ||
		(req.Policy == nil || req.Policy.AllowGatewayTests == nil) && (req.Runtime == nil || req.Runtime.AITimeoutMs == nil) {
		c.JSON(http.StatusBadRequest, model.Error(400, "请提供有效的策略或运行配置"))
		return
	}
	if req.Runtime != nil && req.Runtime.AITimeoutMs != nil && (*req.Runtime.AITimeoutMs < 500 || *req.Runtime.AITimeoutMs > 8000) {
		c.JSON(http.StatusBadRequest, model.Error(400, "AI 检测超时须介于 500 与 8000 毫秒"))
		return
	}
	h.settings.mu.Lock()
	if req.Policy != nil && req.Policy.AllowGatewayTests != nil {
		h.settings.value.Policy.AllowGatewayTests = *req.Policy.AllowGatewayTests
	}
	if req.Runtime != nil && req.Runtime.AITimeoutMs != nil {
		h.settings.value.Runtime.AITimeoutMs = *req.Runtime.AITimeoutMs
	}
	updated := h.settings.value
	h.settings.mu.Unlock()
	c.JSON(http.StatusOK, model.Success(updated))
}
