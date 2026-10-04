package api

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"

	"github.com/chainwise/backend/internal/model"
)

// ==================== 业务后台 CRUD（用户 / 角色 / 数据分级） ====================
//
// 契约来源：frontend/src/api/admin.js
//
//	GET/POST       /api/users
//	GET/PUT/DELETE /api/users/:id
//	GET            /api/roles
//	GET            /api/data-levels
//
// 现状说明：链上三大合约客户端（internal/contract）目前仍是桩实现，尚无真实的
// FISCO BCOS Go-SDK 调用，因此用户与权限数据先由本文件的内存存储承载，保证前端
// 业务后台可完整联调（增删改查闭环）。待合约客户端接入后，应将下列方法替换为
// 链上调用（registerUser / updateRole / updateAccessLevel / deactivateUser），
// 内存存储仅保留为缓存或直接移除。

// demoUser 是业务后台用户视图，字段与前端 mock/data.js 的 mockUsers() 保持一致。
// 注意：响应中永远不返回密码字段。
type demoUser struct {
	ID          int          `json:"id"`
	Username    string       `json:"username"`
	Role        string       `json:"role"`
	DataLevel   string       `json:"dataLevel"`
	Department  string       `json:"department"`
	Permissions []string     `json:"permissions"`
	Scope       accountScope `json:"scope"`

	// password 仅内部保存，不参与 JSON 序列化。
	password string
}

// 合法的角色与密级取值。与链上 AccessControl 合约的 Role / DataLevel 枚举对应：
//
//	Role:      NONE / AUDITOR / OPERATOR / MANAGER / ADMIN
//	DataLevel: PUBLIC / INTERNAL / CONFIDENTIAL / SECRET / TOP_SECRET
//
// 业务侧使用中文角色名与 L1-L4 密级标识（与前端展示、mock 数据一致）。
var demoRoles = []string{roleTeller, roleReviewer, roleAuditor, roleAdmin}

var demoUsernamePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z_-]*$`)

// roleRank 给出各角色可访问的最高密级序号，用于校验「角色-密级」搭配是否合理。
var roleRank = map[string]int{
	roleTeller:   4,
	roleReviewer: 4,
	roleAuditor:  4,
	roleAdmin:    4,
}

// dataLevelMeta 描述四级数据密级，字段与前端 mockDataLevels() 一致。
type dataLevelMeta struct {
	Level  string `json:"level"`
	Desc   string `json:"desc"`
	Fields string `json:"fields"`
	Rank   int    `json:"rank"`
}

var demoDataLevels = []dataLevelMeta{
	{Level: "L1", Desc: "公开数据", Fields: "无", Rank: 1},
	{Level: "L2", Desc: "内部数据", Fields: "基础客户信息", Rank: 2},
	{Level: "L3", Desc: "敏感数据", Fields: "账户余额、交易明细", Rank: 3},
	{Level: "L4", Desc: "高度敏感", Fields: "身份证、卡号、征信", Rank: 4},
}

// userStore 是并发安全的内存用户存储（gin 会并发处理请求）。
type userStore struct {
	mu     sync.RWMutex
	users  []*demoUser
	nextID int
}

var demoUsers = &userStore{}

func init() {
	demoUsers.reset()
}

// reset 以固定的演示数据初始化存储（可重复调用，便于测试隔离）。
func (s *userStore) reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.users = []*demoUser{
		{ID: 1, Username: "admin", Role: roleAdmin, DataLevel: "L4", password: "admin"},
		{ID: 2, Username: "reviewer", Role: roleReviewer, DataLevel: "L3", password: "reviewer"},
		{ID: 3, Username: "teller", Role: roleTeller, DataLevel: "L1", password: "teller"},
		{ID: 4, Username: "auditor", Role: roleAuditor, DataLevel: "L2", password: "auditor"},
	}
	for _, user := range s.users {
		user.Department = defaultDepartment(user.Role)
		user.Scope = defaultScope(user.Role)
	}
	s.nextID = 5
}

// Copy the public view while holding the store lock, so responses and login
// never read a mutable stored user after releasing the lock.
func userSnapshot(u *demoUser) *demoUser {
	return publicIdentity(u)
}

func (s *userStore) authenticate(username, password string) (*demoUser, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, u := range s.users {
		if u.Username == username && u.password == password {
			return userSnapshot(u), true
		}
	}
	return nil, false
}

func (s *userStore) list() []*demoUser {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*demoUser, len(s.users))
	for i, u := range s.users {
		out[i] = userSnapshot(u)
	}
	return out
}

func (s *userStore) find(id int) (*demoUser, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, u := range s.users {
		if u.ID == id {
			return userSnapshot(u), true
		}
	}
	return nil, false
}

func (s *userStore) existsByName(username string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, u := range s.users {
		if u.Username == username {
			return true
		}
	}
	return false
}

func (s *userStore) create(username, role, dataLevel, password, department string, scope accountScope) *demoUser {
	s.mu.Lock()
	defer s.mu.Unlock()

	u := &demoUser{
		ID:         s.nextID,
		Username:   username,
		Role:       role,
		DataLevel:  dataLevel,
		password:   password,
		Department: department, Scope: scope,
	}
	s.nextID++
	s.users = append(s.users, u)
	return userSnapshot(u)
}

func (s *userStore) update(id int, role, dataLevel string) (*demoUser, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, u := range s.users {
		if u.ID == id {
			if role != "" {
				u.Role = role
				u.Scope = scopeFor(role, u.Department)
			}
			if dataLevel != "" {
				u.DataLevel = dataLevel
			}
			return userSnapshot(u), true
		}
	}
	return nil, false
}

func (s *userStore) updateAuthorization(id int, role, level, department string, scope accountScope) (*demoUser, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, user := range s.users {
		if user.ID == id {
			user.Role, user.DataLevel, user.Department, user.Scope = role, level, department, scope
			return userSnapshot(user), true
		}
	}
	return nil, false
}

func validateAccountScope(role, department string, requested *accountScope) (accountScope, string) {
	allowed := scopeFor(role, department)
	if role == roleAdmin {
		if requested != nil && (len(requested.Departments) > 0 || requested.OwnerOnly) {
			return allowed, "系统管理员不能配置客户数据范围"
		}
		return allowed, ""
	}
	if department != "零售业务部" && department != "信贷业务部" {
		return allowed, "所属部门须为零售业务部或信贷业务部"
	}
	if requested == nil {
		return allowed, ""
	}
	if role == roleTeller && !requested.OwnerOnly {
		return allowed, "柜员／客服只能查看本人业务"
	}
	result := accountScope{Departments: []string{}, OwnerOnly: requested.OwnerOnly}
	seen := map[string]bool{}
	for _, department := range requested.Departments {
		valid := false
		for _, candidate := range allowed.Departments {
			if department == candidate {
				valid = true
				break
			}
		}
		if !valid {
			return allowed, "授权部门超出该职责允许的范围"
		}
		if !seen[department] {
			result.Departments = append(result.Departments, department)
			seen[department] = true
		}
	}
	return result, ""
}

func (s *userStore) delete(id int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, u := range s.users {
		if u.ID == id {
			s.users = append(s.users[:i], s.users[i+1:]...)
			return true
		}
	}
	return false
}

// ==================== 参数校验 ====================

func isValidRole(role string) bool {
	_, ok := roleRank[role]
	return ok
}

func isValidDataLevel(level string) bool {
	for _, l := range demoDataLevels {
		if l.Level == level {
			return true
		}
	}
	return false
}

// parseUserID 解析路径中的用户 ID，失败时已写入错误响应并返回 false。
func parseUserID(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, model.Error(400, "invalid user id"))
		return 0, false
	}
	return id, true
}

// ==================== Handler ====================

// DemoListUsers 获取用户列表
// GET /api/users
func (h *Handler) DemoListUsers(c *gin.Context) {
	c.JSON(http.StatusOK, model.Success(demoUsers.list()))
}

// DemoGetUser 获取单个用户
// GET /api/users/:id
func (h *Handler) DemoGetUser(c *gin.Context) {
	id, ok := parseUserID(c)
	if !ok {
		return
	}

	u, found := demoUsers.find(id)
	if !found {
		c.JSON(http.StatusNotFound, model.Error(404, "user not found"))
		return
	}
	c.JSON(http.StatusOK, model.Success(u))
}

// DemoCreateUser 新增用户
// POST /api/users  请求体: { username, role, dataLevel, password? }
func (h *Handler) DemoCreateUser(c *gin.Context) {
	var req struct {
		Username   string        `json:"username"`
		Role       string        `json:"role"`
		DataLevel  string        `json:"dataLevel"`
		Password   string        `json:"password"`
		Department string        `json:"department"`
		Scope      *accountScope `json:"scope"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Error(400, "invalid request body"))
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" {
		c.JSON(http.StatusBadRequest, model.Error(400, "username is required"))
		return
	}
	if !demoUsernamePattern.MatchString(req.Username) {
		c.JSON(http.StatusBadRequest, model.Error(400, "账号须使用英文名称，可包含下划线或连字符"))
		return
	}
	if strings.TrimSpace(req.Password) == "" {
		c.JSON(http.StatusBadRequest, model.Error(400, "请设置登录密码"))
		return
	}
	if !isValidRole(req.Role) {
		c.JSON(http.StatusBadRequest, model.Error(400, "invalid role: "+req.Role))
		return
	}
	if !isValidDataLevel(req.DataLevel) {
		c.JSON(http.StatusBadRequest, model.Error(400, "invalid dataLevel: "+req.DataLevel))
		return
	}
	if req.Department == "" {
		req.Department = defaultDepartment(req.Role)
	}
	if req.Role == roleAdmin {
		req.Department = defaultDepartment(roleAdmin)
	}
	scope, scopeError := validateAccountScope(req.Role, req.Department, req.Scope)
	if scopeError != "" {
		c.JSON(http.StatusBadRequest, model.Error(400, scopeError))
		return
	}
	if demoUsers.existsByName(req.Username) {
		c.JSON(http.StatusConflict, model.Error(409, "username already exists"))
		return
	}

	// TODO(contract): 接入 FISCO BCOS Go-SDK 后改为调用 AccessControl.registerUser
	u := demoUsers.create(req.Username, req.Role, req.DataLevel, req.Password, req.Department, scope)
	c.JSON(http.StatusCreated, model.Success(u))
}

// DemoUpdateUser 更新用户角色 / 密级
// PUT /api/users/:id  请求体: { role?, dataLevel? }
func (h *Handler) DemoUpdateUser(c *gin.Context) {
	id, ok := parseUserID(c)
	if !ok {
		return
	}

	var req struct {
		Role       string        `json:"role"`
		DataLevel  string        `json:"dataLevel"`
		Department string        `json:"department"`
		Scope      *accountScope `json:"scope"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Error(400, "invalid request body"))
		return
	}

	if req.Role == "" && req.DataLevel == "" && req.Department == "" && req.Scope == nil {
		c.JSON(http.StatusBadRequest, model.Error(400, "role or dataLevel is required"))
		return
	}
	if req.Role != "" && !isValidRole(req.Role) {
		c.JSON(http.StatusBadRequest, model.Error(400, "invalid role: "+req.Role))
		return
	}
	if req.DataLevel != "" && !isValidDataLevel(req.DataLevel) {
		c.JSON(http.StatusBadRequest, model.Error(400, "invalid dataLevel: "+req.DataLevel))
		return
	}
	current, found := demoUsers.find(id)
	if !found {
		c.JSON(http.StatusNotFound, model.Error(404, "user not found"))
		return
	}
	if current.Username == "admin" {
		c.JSON(http.StatusForbidden, model.Error(403, "内置系统管理员的职责与客户密级不可变更"))
		return
	}

	// 校验更新后的角色-密级组合仍然合法
	newRole := req.Role
	if newRole == "" {
		newRole = current.Role
	}
	newLevel := req.DataLevel
	if newLevel == "" {
		newLevel = current.DataLevel
	}
	newDepartment := req.Department
	if newDepartment == "" {
		newDepartment = current.Department
	}
	if newRole == roleAdmin {
		newDepartment = defaultDepartment(roleAdmin)
	}
	requestedScope := req.Scope
	if requestedScope == nil && newRole == current.Role && newDepartment == current.Department {
		requestedScope = &current.Scope
	}
	scope, scopeError := validateAccountScope(newRole, newDepartment, requestedScope)
	if scopeError != "" {
		c.JSON(http.StatusBadRequest, model.Error(400, scopeError))
		return
	}

	// TODO(contract): 接入后改为调用 updateRole / updateAccessLevel
	u, found := demoUsers.updateAuthorization(id, newRole, newLevel, newDepartment, scope)
	if !found {
		c.JSON(http.StatusNotFound, model.Error(404, "user not found"))
		return
	}
	c.JSON(http.StatusOK, model.Success(u))
}

// DemoDeleteUser 删除用户
// DELETE /api/users/:id
func (h *Handler) DemoDeleteUser(c *gin.Context) {
	id, ok := parseUserID(c)
	if !ok {
		return
	}

	// 保护内置管理员账号，避免演示过程中把后台锁死
	if u, found := demoUsers.find(id); found && u.Username == "admin" {
		c.JSON(http.StatusForbidden, model.Error(403, "the built-in admin account cannot be deleted"))
		return
	}

	// TODO(contract): 接入后改为调用 AccessControl.deactivateUser（链上仅停用不删除）
	if !demoUsers.delete(id) {
		c.JSON(http.StatusNotFound, model.Error(404, "user not found"))
		return
	}
	c.JSON(http.StatusOK, model.Success(gin.H{"id": id, "deleted": true}))
}

// DemoListRoles 获取角色列表
// GET /api/roles
func (h *Handler) DemoListRoles(c *gin.Context) {
	type roleItem struct {
		Name             string       `json:"name"`
		MaxAccessLevel   string       `json:"maxAccessLevel"`
		ChainRoleOrdinal int          `json:"chainRoleOrdinal"`
		Permissions      []string     `json:"permissions"`
		Scope            accountScope `json:"scope"`
	}

	// 角色名与链上枚举序号的对应关系：
	// NONE=0, AUDITOR=1, OPERATOR=2, MANAGER=3, ADMIN=4
	ordinal := map[string]int{
		roleAuditor:  1, // AUDITOR
		roleTeller:   2, // OPERATOR
		roleReviewer: 3, // MANAGER
		roleAdmin:    4, // ADMIN (customer read is a separate permission)
	}

	items := make([]roleItem, 0, len(demoRoles))
	for _, r := range demoRoles {
		items = append(items, roleItem{
			Name:             r,
			MaxAccessLevel:   "L" + strconv.Itoa(roleRank[r]),
			ChainRoleOrdinal: ordinal[r],
			Permissions:      append([]string{}, rolePermissions[r]...),
			Scope:            defaultScope(r),
		})
	}
	c.JSON(http.StatusOK, model.Success(items))
}

// DemoListDataLevels 获取数据分级配置
// GET /api/data-levels
func (h *Handler) DemoListDataLevels(c *gin.Context) {
	c.JSON(http.StatusOK, model.Success(demoDataLevels))
}

// ==================== 内部工具 ====================

// levelRank 返回密级标识（L1-L4）对应的序号，未知密级返回 0。
func levelRank(level string) int {
	for _, l := range demoDataLevels {
		if l.Level == level {
			return l.Rank
		}
	}
	return 0
}
