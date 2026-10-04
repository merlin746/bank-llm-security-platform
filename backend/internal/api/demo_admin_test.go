package api

import (
	"net/http"
	"testing"
)

// ==================== 用户列表 ====================

func TestDemoListUsers(t *testing.T) {
	r := newTestEngine(t)

	w, resp := doRequest(t, r, http.MethodGet, "/api/users", nil)
	assertSuccess(t, w, resp)

	var users []demoUser
	decodeData(t, resp, &users)

	if len(users) != 4 {
		t.Fatalf("expected 4 seeded users, got %d", len(users))
	}
	if users[0].Username != "admin" {
		t.Errorf("expected first user to be admin, got %s", users[0].Username)
	}
	if users[0].Role != "系统管理员" || users[0].DataLevel != "L4" {
		t.Errorf("unexpected admin role/level: %s / %s", users[0].Role, users[0].DataLevel)
	}
}

// TestDemoListUsersNeverExposesPassword 密码字段绝不能出现在响应中。
func TestDemoListUsersNeverExposesPassword(t *testing.T) {
	r := newTestEngine(t)

	w, _ := doRequest(t, r, http.MethodGet, "/api/users", nil)
	body := w.Body.String()

	if contains(body, "password") {
		t.Errorf("response must not contain password field, body=%s", body)
	}
}

// ==================== 创建用户 ====================

func TestDemoCreateUser(t *testing.T) {
	r := newTestEngine(t)

	payload := map[string]string{
		"username":  "audit_one",
		"role":      "风控审核员",
		"dataLevel": "L3",
		"password":  "secret",
	}
	w, resp := doRequest(t, r, http.MethodPost, "/api/users", payload)
	assertStatus(t, w, http.StatusCreated)
	if resp.Code != 0 {
		t.Fatalf("expected business code 0, got %d (%s)", resp.Code, resp.Message)
	}

	var created demoUser
	decodeData(t, resp, &created)

	if created.ID != 5 {
		t.Errorf("expected auto-increment id 5, got %d", created.ID)
	}
	if created.Username != "audit_one" {
		t.Errorf("unexpected username: %s", created.Username)
	}

	// 列表应包含新用户
	w, resp = doRequest(t, r, http.MethodGet, "/api/users", nil)
	assertSuccess(t, w, resp)

	var users []demoUser
	decodeData(t, resp, &users)
	if len(users) != 5 {
		t.Fatalf("expected 5 users after create, got %d", len(users))
	}
}

func TestDemoCreateUserRejectsEmptyUsername(t *testing.T) {
	r := newTestEngine(t)

	w, _ := doRequest(t, r, http.MethodPost, "/api/users", map[string]string{
		"username":  "   ",
		"role":      "柜员／客服",
		"dataLevel": "L2",
	})
	assertStatus(t, w, http.StatusBadRequest)
}

func TestDemoCreateUserRejectsInvalidRole(t *testing.T) {
	r := newTestEngine(t)

	w, _ := doRequest(t, r, http.MethodPost, "/api/users", map[string]string{
		"username":  "x_01",
		"role":      "超级系统管理员",
		"dataLevel": "L2",
	})
	assertStatus(t, w, http.StatusBadRequest)
}

func TestDemoCreateUserRejectsInvalidDataLevel(t *testing.T) {
	r := newTestEngine(t)

	w, _ := doRequest(t, r, http.MethodPost, "/api/users", map[string]string{
		"username":  "x_01",
		"role":      "柜员／客服",
		"dataLevel": "L9",
	})
	assertStatus(t, w, http.StatusBadRequest)
}

// Role actions and explicitly assigned clearance remain independent.
func TestDemoCreateUserAllowsExplicitClearanceWithoutWideningRoleActions(t *testing.T) {
	r := newTestEngine(t)

	w, resp := doRequest(t, r, http.MethodPost, "/api/users", map[string]string{
		"username":  "teller_extra",
		"password":  "teller_extra",
		"role":      "柜员／客服",
		"dataLevel": "L4",
	})
	assertStatus(t, w, http.StatusCreated)
	var user demoUser
	decodeData(t, resp, &user)
	if user.DataLevel != "L4" || !user.Scope.OwnerOnly || hasPermission(&user, "admin.manage") {
		t.Errorf("clearance widened role actions or ownership: %+v", user)
	}
}

func TestDemoCreateUserRejectsDuplicateUsername(t *testing.T) {
	r := newTestEngine(t)

	w, _ := doRequest(t, r, http.MethodPost, "/api/users", map[string]string{
		"username":  "admin",
		"password":  "admin",
		"role":      "系统管理员",
		"dataLevel": "L4",
	})
	assertStatus(t, w, http.StatusConflict)
}

func TestDemoCreateUserRejectsMalformedJSON(t *testing.T) {
	r := newTestEngine(t)

	// 直接用非法 JSON 触发 ShouldBindJSON 失败
	w, _ := doRequestRaw(t, r, http.MethodPost, "/api/users", `{"username":`)
	assertStatus(t, w, http.StatusBadRequest)
}

// ==================== 查询单个用户 ====================

func TestDemoGetUser(t *testing.T) {
	r := newTestEngine(t)

	w, resp := doRequest(t, r, http.MethodGet, "/api/users/2", nil)
	assertSuccess(t, w, resp)

	var u demoUser
	decodeData(t, resp, &u)
	if u.Username != "reviewer" {
		t.Errorf("expected reviewer, got %s", u.Username)
	}
}

func TestDemoGetUserNotFound(t *testing.T) {
	r := newTestEngine(t)

	w, _ := doRequest(t, r, http.MethodGet, "/api/users/9999", nil)
	assertStatus(t, w, http.StatusNotFound)
}

func TestDemoGetUserInvalidID(t *testing.T) {
	r := newTestEngine(t)

	w, _ := doRequest(t, r, http.MethodGet, "/api/users/abc", nil)
	assertStatus(t, w, http.StatusBadRequest)
}

// ==================== 更新用户 ====================

func TestDemoUpdateUserRole(t *testing.T) {
	r := newTestEngine(t)

	w, resp := doRequest(t, r, http.MethodPut, "/api/users/3", map[string]string{
		"role":      "风控审核员",
		"dataLevel": "L3",
	})
	assertSuccess(t, w, resp)

	var updated demoUser
	decodeData(t, resp, &updated)
	if updated.Role != "风控审核员" || updated.DataLevel != "L3" {
		t.Errorf("update not applied: %s / %s", updated.Role, updated.DataLevel)
	}

	// 变更必须持久化
	w, resp = doRequest(t, r, http.MethodGet, "/api/users/3", nil)
	assertSuccess(t, w, resp)
	decodeData(t, resp, &updated)
	if updated.Role != "风控审核员" {
		t.Errorf("update was not persisted, got role %s", updated.Role)
	}
}

func TestDemoUpdateUserRejectsEmptyPayload(t *testing.T) {
	r := newTestEngine(t)

	w, _ := doRequest(t, r, http.MethodPut, "/api/users/3", map[string]string{})
	assertStatus(t, w, http.StatusBadRequest)
}

func TestDemoUpdateUserNotFound(t *testing.T) {
	r := newTestEngine(t)

	w, _ := doRequest(t, r, http.MethodPut, "/api/users/9999", map[string]string{
		"role": "柜员／客服",
	})
	assertStatus(t, w, http.StatusNotFound)
}

func TestDemoUpdateBuiltinAdminRoleIsProtected(t *testing.T) {
	r := newTestEngine(t)

	w, _ := doRequest(t, r, http.MethodPut, "/api/users/1", map[string]string{
		"role": "柜员／客服",
	})
	assertStatus(t, w, http.StatusForbidden)
}

// ==================== 删除用户 ====================

func TestDemoDeleteUser(t *testing.T) {
	r := newTestEngine(t)

	w, resp := doRequest(t, r, http.MethodDelete, "/api/users/3", nil)
	assertSuccess(t, w, resp)

	w, resp = doRequest(t, r, http.MethodGet, "/api/users", nil)
	assertSuccess(t, w, resp)

	var users []demoUser
	decodeData(t, resp, &users)
	if len(users) != 3 {
		t.Fatalf("expected 3 users after delete, got %d", len(users))
	}
}

// TestDemoDeleteBuiltinAdminIsForbidden 内置 admin 账号不可删除，避免锁死后台。
func TestDemoDeleteBuiltinAdminIsForbidden(t *testing.T) {
	r := newTestEngine(t)

	w, _ := doRequest(t, r, http.MethodDelete, "/api/users/1", nil)
	assertStatus(t, w, http.StatusForbidden)
}

func TestDemoDeleteUserNotFound(t *testing.T) {
	r := newTestEngine(t)

	w, _ := doRequest(t, r, http.MethodDelete, "/api/users/9999", nil)
	assertStatus(t, w, http.StatusNotFound)
}

// ==================== 角色与数据分级 ====================

func TestDemoListRoles(t *testing.T) {
	r := newTestEngine(t)

	w, resp := doRequest(t, r, http.MethodGet, "/api/roles", nil)
	assertSuccess(t, w, resp)

	var roles []struct {
		Name             string `json:"name"`
		MaxAccessLevel   string `json:"maxAccessLevel"`
		ChainRoleOrdinal int    `json:"chainRoleOrdinal"`
	}
	decodeData(t, resp, &roles)

	if len(roles) != 4 {
		t.Fatalf("expected 4 roles, got %d", len(roles))
	}

	byName := make(map[string]string, len(roles))
	for _, role := range roles {
		byName[role.Name] = role.MaxAccessLevel
	}
	if byName["系统管理员"] != "L4" {
		t.Errorf("系统管理员 should map to L4, got %s", byName["系统管理员"])
	}
	if byName["柜员／客服"] != "L4" {
		t.Errorf("clearance configuration should remain independent, got %s", byName["柜员／客服"])
	}
	if byName["审计人员"] != "L4" {
		t.Errorf("审计人员 clearance configuration should allow L4, got %s", byName["审计人员"])
	}
	for _, role := range roles {
		if role.Name == "审计人员" && role.ChainRoleOrdinal != 1 {
			t.Errorf("审计人员 chain ordinal = %d, want AUDITOR=1", role.ChainRoleOrdinal)
		}
	}
}

func TestDemoListDataLevels(t *testing.T) {
	r := newTestEngine(t)

	w, resp := doRequest(t, r, http.MethodGet, "/api/data-levels", nil)
	assertSuccess(t, w, resp)

	var levels []dataLevelMeta
	decodeData(t, resp, &levels)

	if len(levels) != 4 {
		t.Fatalf("expected 4 data levels, got %d", len(levels))
	}
	if levels[0].Level != "L1" || levels[3].Level != "L4" {
		t.Errorf("unexpected level ordering: %s ... %s", levels[0].Level, levels[3].Level)
	}
	for i, l := range levels {
		if l.Rank != i+1 {
			t.Errorf("level %s expected rank %d, got %d", l.Level, i+1, l.Rank)
		}
	}
}
