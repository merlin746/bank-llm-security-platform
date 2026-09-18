package api

import (
	"testing"
)

// frontendContractRoutes 是前端实际调用的全部路由（含刻意重复声明的成员2联调路由）。
//
// 依据：
//   - frontend/src/api/auth.js
//   - frontend/src/api/gateway.js
//   - frontend/src/api/stats.js
//   - frontend/src/api/audit.js
//   - frontend/src/api/admin.js
//   - frontend/README.md 的接口契约表
//
// 该表同时作为「契约回归测试」：任何一条路由缺失都会导致前端在
// VITE_USE_MOCK=false 联调模式下 404，因此必须全部注册。
var frontendContractRoutes = []struct {
	Method string
	Path   string
	Source string
}{
	{"POST", "/api/auth/login", "auth.js"},
	{"POST", "/api/gateway/attack-test", "gateway.js"},
	{"POST", "/api/gateway/access-test", "gateway.js"},
	{"GET", "/api/stats/overview", "stats.js"},
	{"GET", "/api/stats/trend", "stats.js"},
	{"GET", "/api/stats/risk-distribution", "stats.js"},
	{"GET", "/api/stats/high-risk-users", "stats.js"},
	{"GET", "/api/audit/topology", "audit.js"},
	{"GET", "/api/audit/alerts", "audit.js"},
	{"GET", "/api/users", "admin.js"},
	{"POST", "/api/users", "admin.js"},
	{"PUT", "/api/users/:id", "admin.js"},
	{"DELETE", "/api/users/:id", "admin.js"},
	{"GET", "/api/roles", "admin.js"},
	{"GET", "/api/data-levels", "admin.js"},
}

// TestFrontendContractRoutesAreRegistered 断言前端依赖的所有路由都已注册。
func TestFrontendContractRoutesAreRegistered(t *testing.T) {
	r := newTestEngine(t)

	registered := make(map[string]bool)
	for _, route := range r.Routes() {
		registered[route.Method+" "+route.Path] = true
	}

	for _, want := range frontendContractRoutes {
		key := want.Method + " " + want.Path
		if !registered[key] {
			t.Errorf("route not registered: %s (required by frontend/src/api/%s)", key, want.Source)
		}
	}
}

// TestCoreV1RoutesAreRegistered 断言后端核心 /api/v1 接口已注册。
func TestCoreV1RoutesAreRegistered(t *testing.T) {
	r := newTestEngine(t)

	registered := make(map[string]bool)
	for _, route := range r.Routes() {
		registered[route.Method+" "+route.Path] = true
	}

	required := []string{
		"GET /api/v1/health",
		"GET /api/v1/audit/stats",
		"GET /api/v1/audit/requests",
		"GET /api/v1/audit/requests/:requestId",
		"GET /api/v1/audit/anomalies",
		"GET /api/v1/permission/users/:address",
		"POST /api/v1/permission/check-access",
		"GET /api/v1/policy/active",
		"GET /api/v1/policy/rules",
		"POST /api/v1/ai/prompt/detect",
		"POST /api/v1/ai/output/desensitize",
		"POST /api/v1/ai/risk/score",
	}

	for _, key := range required {
		if !registered[key] {
			t.Errorf("core route not registered: %s", key)
		}
	}
}

// TestNoDuplicateRoutePanics 确认注册过程本身不产生路由冲突（会 panic）。
func TestNoDuplicateRoutePanics(t *testing.T) {
	defer func() {
		if rec := recover(); rec != nil {
			t.Fatalf("route registration panicked: %v", rec)
		}
	}()

	_ = newTestEngine(t)
}
