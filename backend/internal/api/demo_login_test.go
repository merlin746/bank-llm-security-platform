package api

import (
	"fmt"
	"net/http"
	"testing"
)

type demoLoginResponse struct {
	Token string   `json:"token"`
	User  demoUser `json:"user"`
}

func TestDemoLoginDistinctAccountsForEachClearance(t *testing.T) {
	r := newTestEngine(t)
	for _, account := range []struct {
		username string
		role     string
		level    string
	}{
		{"auditor", "审计人员", "L2"},
		{"teller", "柜员／客服", "L1"},
		{"reviewer", "风控审核员", "L3"},
		{"admin", "系统管理员", "L4"},
	} {
		t.Run(account.username, func(t *testing.T) {
			w, response := doRequest(t, r, http.MethodPost, "/api/auth/login", map[string]string{
				"username": "  " + account.username + "\t", "password": account.username,
			})
			assertSuccess(t, w, response)
			var login demoLoginResponse
			decodeData(t, response, &login)
			if login.Token == "" || login.User.Username != account.username || login.User.Role != account.role || login.User.DataLevel != account.level {
				t.Errorf("account returned incorrect identity: %+v", login)
			}
			if contains(w.Body.String(), "password") {
				t.Error("login response exposes a password")
			}
		})
	}
}

func TestDemoLoginRejectsWrongCredentialsUniformly(t *testing.T) {
	r := newTestEngine(t)
	for _, credentials := range []map[string]string{
		{"username": "does_not_exist", "password": "anything"},
		{"username": "reviewer", "password": "wrong"},
		{"username": "reviewer", "password": "admin"},
		{"username": "reviewer", "password": " reviewer "},
		{"username": "ADMIN", "password": "admin"},
	} {
		w, response := doRequest(t, r, http.MethodPost, "/api/auth/login", credentials)
		assertStatus(t, w, http.StatusUnauthorized)
		if response.Code != 401 || response.Msg != "账号或密码错误" || response.Message != "账号或密码错误" || len(response.Data) != 0 {
			t.Errorf("credential rejection is inconsistent: %+v", response)
		}
	}
}

func TestDemoLoginRejectsBlankOrMalformedCredentials(t *testing.T) {
	r := newTestEngine(t)
	for _, credentials := range []map[string]string{
		{},
		{"username": "reviewer"},
		{"password": "reviewer"},
		{"username": "   ", "password": "reviewer"},
		{"username": "reviewer", "password": ""},
		{"username": "reviewer", "password": "   "},
	} {
		w, response := doRequest(t, r, http.MethodPost, "/api/auth/login", credentials)
		assertStatus(t, w, http.StatusBadRequest)
		if response.Code != 400 || response.Msg != "请输入账号和密码" {
			t.Errorf("invalid credentials returned incorrect error: %+v", response)
		}
	}
	w, _ := doRequestRaw(t, r, http.MethodPost, "/api/auth/login", `{"username":`)
	assertStatus(t, w, http.StatusBadRequest)
}

func TestDemoLoginFollowsUserCreationRoleUpdateAndDeletion(t *testing.T) {
	r := newTestEngine(t)
	w, response := doRequest(t, r, http.MethodPost, "/api/users", map[string]string{
		"username": "new_viewer", "password": " exact secret ", "role": "审计人员", "dataLevel": "L1",
	})
	assertStatus(t, w, http.StatusCreated)
	var created demoUser
	decodeData(t, response, &created)
	credentials := map[string]string{"username": "new_viewer", "password": " exact secret "}
	w, response = doRequest(t, r, http.MethodPost, "/api/auth/login", credentials)
	assertSuccess(t, w, response)
	var login demoLoginResponse
	decodeData(t, response, &login)
	if login.User.Role != "审计人员" || login.User.DataLevel != "L1" {
		t.Errorf("created account did not use stored clearance: %+v", login.User)
	}
	w, _ = doRequest(t, r, http.MethodPost, "/api/auth/login", map[string]string{"username": "new_viewer", "password": "exact secret"})
	assertStatus(t, w, http.StatusUnauthorized)
	w, response = doRequest(t, r, http.MethodPut, fmt.Sprintf("/api/users/%d", created.ID), map[string]string{"role": "风控审核员", "dataLevel": "L3"})
	assertSuccess(t, w, response)
	w, response = doRequest(t, r, http.MethodPost, "/api/auth/login", credentials)
	assertSuccess(t, w, response)
	decodeData(t, response, &login)
	if login.User.Role != "风控审核员" || login.User.DataLevel != "L3" {
		t.Errorf("login returned obsolete role after CRUD update: %+v", login.User)
	}
	w, response = doRequest(t, r, http.MethodDelete, fmt.Sprintf("/api/users/%d", created.ID), nil)
	assertSuccess(t, w, response)
	w, response = doRequest(t, r, http.MethodPost, "/api/auth/login", credentials)
	assertStatus(t, w, http.StatusUnauthorized)
	if response.Msg != "账号或密码错误" {
		t.Errorf("deleted account returned incorrect login error: %+v", response)
	}
}

func TestTellerCannotRunSecuritySimulations(t *testing.T) {
	r := newTestEngine(t)
	token := loginToken(t, r, "teller")
	for _, level := range []string{"L1", "L2", "L3", "L4"} {
		w, _ := doRequestWithToken(t, r, http.MethodPost, "/api/gateway/access-test", map[string]string{"role": "系统管理员", "dataLevel": level}, token)
		assertStatus(t, w, http.StatusForbidden)
	}
}
func TestUserStoreReturnsStableSnapshots(t *testing.T) {
	store := &userStore{}
	store.reset()
	found, _ := store.find(2)
	listed := store.list()[1]
	authenticated, _ := store.authenticate("reviewer", "reviewer")
	updated, _ := store.update(2, "柜员／客服", "L2")
	store.update(2, "审计人员", "L1")
	for _, snapshot := range []*demoUser{found, listed, authenticated} {
		if snapshot.Role != "风控审核员" || snapshot.DataLevel != "L3" || snapshot.password != "" {
			t.Errorf("store response changed with a later update or exposed password: %+v", snapshot)
		}
	}
	if updated.Role != "柜员／客服" || updated.DataLevel != "L2" {
		t.Errorf("update result is not a stable snapshot: %+v", updated)
	}
}
