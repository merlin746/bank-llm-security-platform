package api

import (
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/chainwise/backend/internal/model"
	"github.com/gin-gonic/gin"
)

func (h *Handler) currentDayRequests(c *gin.Context) []demoRequest {
	today := demoTimestamp(time.Now())[:10]
	items := []demoRequest{}
	for _, request := range h.demoAudit.visibleRequests(currentUser(c), false) {
		if len(request.Time) >= 10 && request.Time[:10] == today {
			items = append(items, request)
		}
	}
	return items
}

func (h *Handler) DemoStatsOverview(c *gin.Context) {
	requests := h.currentDayRequests(c)
	blocked := 0
	users := map[string]bool{}
	for _, request := range requests {
		if request.Verdict == "block" {
			blocked++
			users[request.Owner] = true
		}
	}
	rate := 0.0
	if len(requests) > 0 {
		rate = float64(blocked) * 100 / float64(len(requests))
	}
	c.JSON(http.StatusOK, model.Success(gin.H{"totalRequests": len(requests), "blockedToday": blocked, "blockRate": rate, "highRiskUsers": len(users)}))
}

func (h *Handler) DemoStatsTrend(c *gin.Context) {
	items := make([]gin.H, 6)
	for i := range items {
		items[i] = gin.H{"time": time.Date(2000, 1, 1, i*4, 0, 0, 0, time.UTC).Format("15:04"), "blocked": 0, "passed": 0}
	}
	for _, request := range h.currentDayRequests(c) {
		if len(request.Time) < 13 {
			continue
		}
		hour, err := strconv.Atoi(request.Time[11:13])
		if err != nil || hour < 0 || hour > 23 {
			continue
		}
		field := "passed"
		if request.Verdict == "block" {
			field = "blocked"
		}
		items[hour/4][field] = items[hour/4][field].(int) + 1
	}
	c.JSON(http.StatusOK, model.Success(items))
}

func (h *Handler) DemoStatsRiskDistribution(c *gin.Context) {
	counts := map[string]int{"Prompt 注入": 0, "越权访问": 0}
	for _, request := range h.currentDayRequests(c) {
		if request.Verdict != "block" {
			continue
		}
		if request.Kind == "attack" || request.Kind == "business" {
			counts["Prompt 注入"]++
		} else if request.Kind == "access" {
			counts["越权访问"]++
		}
	}
	c.JSON(http.StatusOK, model.Success([]gin.H{{"type": "Prompt 注入", "value": counts["Prompt 注入"]}, {"type": "越权访问", "value": counts["越权访问"]}}))
}

func (h *Handler) DemoStatsHighRiskUsers(c *gin.Context) {
	byUser := map[string]gin.H{}
	for _, request := range h.currentDayRequests(c) {
		if request.Verdict != "block" {
			continue
		}
		if _, seen := byUser[request.Owner]; seen {
			continue
		}
		action := "Prompt 注入被拦截"
		if request.Kind == "access" {
			action = "超出客户密级访问被拦截"
		}
		byUser[request.Owner] = gin.H{"name": request.Owner, "role": request.Role, "riskScore": 90, "lastAction": action, "level": "高"}
	}
	items := []gin.H{}
	for _, item := range byUser {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i]["name"].(string) < items[j]["name"].(string) })
	c.JSON(http.StatusOK, model.Success(items))
}
