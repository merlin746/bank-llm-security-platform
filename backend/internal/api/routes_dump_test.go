package api

import (
	"sort"
	"testing"
)

// expectedRouteCount 是后端对外暴露的路由总数。
//
// 该断言的价值：新增或删减接口时会立即失败，提醒同步更新
// 《API 接口规范文档》(docs/API接口规范文档.md)，避免文档与代码漂移。
const expectedRouteCount = 28

// TestRouteInventoryMatchesSpec 断言路由总数，并输出完整清单便于人工核对。
//
// 查看清单：go test ./internal/api/ -run TestRouteInventory -v
func TestRouteInventoryMatchesSpec(t *testing.T) {
	r := newTestEngine(t)

	routes := r.Routes()
	lines := make([]string, 0, len(routes))
	seen := make(map[string]bool, len(routes))

	for _, route := range routes {
		key := route.Method + " " + route.Path

		if seen[key] {
			t.Errorf("duplicate route registered: %s", key)
		}
		seen[key] = true
		lines = append(lines, key)
	}
	sort.Strings(lines)

	t.Logf("registered routes (%d):", len(lines))
	for _, line := range lines {
		t.Logf("  %s", line)
	}

	if len(lines) != expectedRouteCount {
		t.Errorf("route count changed: got %d, want %d.\n"+
			"Update expectedRouteCount and docs/API接口规范文档.md together.",
			len(lines), expectedRouteCount)
	}
}
