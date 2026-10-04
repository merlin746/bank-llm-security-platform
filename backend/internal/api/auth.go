package api

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/chainwise/backend/internal/model"
	"github.com/gin-gonic/gin"
)

type accountScope struct {
	Departments []string `json:"departments"`
	OwnerOnly   bool     `json:"ownerOnly"`
}

const (
	roleTeller   = "柜员／客服"
	roleReviewer = "风控审核员"
	roleAuditor  = "审计人员"
	roleAdmin    = "系统管理员"
)

func defaultScope(role string) accountScope {
	return scopeFor(role, defaultDepartment(role))
}

func scopeFor(role, department string) accountScope {
	switch role {
	case roleTeller:
		return accountScope{Departments: []string{department}, OwnerOnly: true}
	case roleReviewer:
		return accountScope{Departments: []string{department}}
	case roleAuditor:
		return accountScope{Departments: []string{"零售业务部", "信贷业务部"}}
	default:
		return accountScope{Departments: []string{}}
	}
}

func defaultDepartment(role string) string {
	switch role {
	case roleTeller:
		return "零售业务部"
	case roleReviewer, roleAuditor:
		return "零售业务部"
	default:
		return "平台运维部"
	}
}

func businessDepartment(u *demoUser) string {
	if len(u.Scope.Departments) > 0 {
		return u.Scope.Departments[0]
	}
	return ""
}

// Roles grant actions. Clearance and an explicit department/owner scope grant
// records independently; the admin role never grants customer-data access.
var rolePermissions = map[string][]string{
	roleTeller:   {"business.read", "business.submit"},
	roleReviewer: {"business.read", "dashboard.read", "alerts.read", "risk.review", "simulation.run"},
	roleAuditor:  {"business.read", "alerts.read", "audit.read"},
	roleAdmin:    {"admin.manage"},
}

func publicIdentity(u *demoUser) *demoUser {
	out := *u
	out.password = ""
	out.Scope = accountScope{Departments: []string{}, OwnerOnly: u.Scope.OwnerOnly || u.Role == roleTeller}
	for _, department := range u.Scope.Departments {
		for _, allowed := range scopeFor(u.Role, u.Department).Departments {
			if department == allowed {
				out.Scope.Departments = append(out.Scope.Departments, department)
				break
			}
		}
	}
	out.Permissions = append([]string{}, rolePermissions[u.Role]...)
	return &out
}

func hasPermission(u *demoUser, permission string) bool {
	if u == nil {
		return false
	}
	for _, p := range rolePermissions[u.Role] {
		if p == permission {
			return true
		}
	}
	return false
}

type authSession struct {
	UserID    int
	ExpiresAt time.Time
}
type sessionStore struct {
	mu       sync.Mutex
	sessions map[string]authSession
}

func newSessionStore() *sessionStore { return &sessionStore{sessions: map[string]authSession{}} }
func (s *sessionStore) issue(userID int) (string, error) {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return "", err
	}
	token := hex.EncodeToString(secret)
	s.mu.Lock()
	defer s.mu.Unlock()
	for key, session := range s.sessions {
		if time.Now().After(session.ExpiresAt) {
			delete(s.sessions, key)
		}
	}
	s.sessions[token] = authSession{UserID: userID, ExpiresAt: time.Now().Add(8 * time.Hour)}
	return token, nil
}
func (s *sessionStore) resolve(token string) (*demoUser, bool) {
	s.mu.Lock()
	session, exists := s.sessions[token]
	if exists && time.Now().After(session.ExpiresAt) {
		delete(s.sessions, token)
		exists = false
	}
	s.mu.Unlock()
	if !exists {
		return nil, false
	}
	// Re-read the account on every request: deletion, role and clearance changes
	// take effect immediately even for tokens issued before the change.
	u, exists := demoUsers.find(session.UserID)
	if !exists {
		return nil, false
	}
	return publicIdentity(u), true
}

func currentUser(c *gin.Context) *demoUser {
	value, _ := c.Get("identity")
	u, _ := value.(*demoUser)
	return u
}

func requiredPermissions(method, path string) []string {
	if path == "/api/auth/me" || path == "/api/auth/logout" {
		return []string{"authenticated"}
	}
	switch path {
	case "/api/business/requests":
		if method == http.MethodPost {
			return []string{"business.submit"}
		}
		return []string{"business.read"}
	case "/api/business/requests/:requestId", "/api/workspace/requests":
		return []string{"business.read"}
	case "/api/stats/overview", "/api/stats/trend", "/api/stats/risk-distribution", "/api/stats/high-risk-users":
		return []string{"dashboard.read"}
	case "/api/audit/alerts", "/api/audit/alerts/:id":
		return []string{"alerts.read"}
	case "/api/audit/requests":
		return []string{"audit.read", "alerts.read"}
	case "/api/audit/requests/:requestId":
		return []string{"audit.read", "alerts.read", "simulation.run"}
	case "/api/audit/topology", "/api/v1/audit/stats", "/api/v1/audit/requests", "/api/v1/audit/requests/:requestId", "/api/v1/audit/anomalies":
		return []string{"audit.read"}
	case "/api/risk/reviews":
		return []string{"risk.review"}
	case "/api/gateway/attack-test", "/api/gateway/access-test", "/api/v1/ai/prompt/detect", "/api/v1/ai/output/desensitize", "/api/v1/ai/risk/score":
		return []string{"simulation.run"}
	case "/api/users", "/api/users/:id", "/api/roles", "/api/data-levels", "/api/admin/settings", "/api/system/settings", "/api/v1/permission/users/:address", "/api/v1/policy/active", "/api/v1/policy/rules", "/api/v1/permission/check-access":
		return []string{"admin.manage"}
	}
	return nil // New endpoints remain inaccessible until explicitly authorized.
}

func (h *Handler) protectAPI(c *gin.Context) {
	if !strings.HasPrefix(c.Request.URL.Path, "/api/") {
		c.Next()
		return
	}
	path := c.FullPath()
	if path == "/api/auth/login" || path == "/api/v1/health" {
		c.Next()
		return
	}
	parts := strings.Fields(c.GetHeader("Authorization"))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		c.AbortWithStatusJSON(http.StatusUnauthorized, model.Error(401, "请先登录"))
		return
	}
	u, valid := h.sessions.resolve(parts[1])
	if !valid {
		c.AbortWithStatusJSON(http.StatusUnauthorized, model.Error(401, "登录已失效，请重新登录"))
		return
	}
	c.Set("identity", u)
	for _, permission := range requiredPermissions(c.Request.Method, path) {
		if permission == "authenticated" || hasPermission(u, permission) {
			c.Next()
			return
		}
	}
	c.AbortWithStatusJSON(http.StatusForbidden, model.Error(403, "当前角色未获此项操作授权"))
}

func (h *Handler) DemoCurrentUser(c *gin.Context) {
	c.JSON(http.StatusOK, model.Success(gin.H{"user": currentUser(c)}))
}
func (h *Handler) DemoLogout(c *gin.Context) {
	token := strings.Fields(c.GetHeader("Authorization"))[1]
	h.sessions.mu.Lock()
	delete(h.sessions.sessions, token)
	h.sessions.mu.Unlock()
	c.JSON(http.StatusOK, model.Success(gin.H{"loggedOut": true}))
}

func canReadData(u *demoUser, owner, department, dataLevel string) bool {
	if u == nil || !hasPermission(u, "business.read") || levelRank(dataLevel) == 0 || levelRank(dataLevel) > levelRank(u.DataLevel) {
		return false
	}
	scope := publicIdentity(u).Scope
	if scope.OwnerOnly && owner != u.Username {
		return false
	}
	for _, allowed := range scope.Departments {
		if department != "" && department == allowed {
			return true
		}
	}
	return false
}

func canReadRequest(u *demoUser, request demoRequest) bool {
	return canReadData(u, request.Owner, request.Department, request.DataLevel)
}

func canReadAlert(u *demoUser, alert demoAlert) bool {
	return canReadData(u, alert.Owner, alert.Department, alert.DataLevel)
}

func denyUnscopedChainData(c *gin.Context) {
	c.JSON(http.StatusForbidden, model.Error(403, "链上演示接口尚未实现授权范围校验，暂不开放访问"))
}
