 // SPDX-License-Identifier: MIT
 pragma solidity ^0.8.0;
 
 /**
  * @title AccessControl
  * @notice 分级权限准入合约：角色校验、数据密级匹配、滑动时间窗口限流
  * @dev 部署于 FISCO BCOS v3.x 联盟链
  */
 contract AccessControl {
     // ==================== 类型定义 ====================
 
     /// @notice 用户角色枚举
     enum Role {
         NONE,        // 无权限
         AUDITOR,     // 审计员
         OPERATOR,    // 普通操作员
         MANAGER,     // 经理
         ADMIN        // 系统管理员
     }
 
     /// @notice 数据密级枚举
     enum DataLevel {
         PUBLIC,      // 公开
         INTERNAL,    // 内部
         CONFIDENTIAL,// 机密
         SECRET,      // 秘密
         TOP_SECRET   // 绝密
     }
 
     /// @notice 用户权限结构体
     struct UserPermission {
         Role role;                  // 角色
         DataLevel maxAccessLevel;   // 可访问最高密级
         bool active;                // 是否启用
     }
 
     /// @notice 频次记录结构体（滑动窗口）
     struct RateRecord {
         uint256[] timestamps;       // 时间戳数组（秒）
     }
 
     // ==================== 状态变量 ====================
 
     address public owner;                       // 合约拥有者
     mapping(address => UserPermission) public users;         // 用户地址 => 权限
     mapping(address => RateRecord) private rateRecords;      // 用户地址 => 调用频次
 
     uint256 public windowSeconds;               // 滑动窗口时长（秒）
     uint256 public maxRequestsPerWindow;         // 窗口内最大请求数
 
     // ==================== 事件 ====================
 
     event UserRegistered(address indexed user, Role role, DataLevel maxLevel);
     event UserDeactivated(address indexed user);
     event UserReactivated(address indexed user);
     event RoleUpdated(address indexed user, Role newRole);
     event AccessLevelUpdated(address indexed user, DataLevel newLevel);
     event AccessDenied(address indexed user, string reason);
     event RateLimitUpdated(uint256 windowSeconds, uint256 maxRequests);
 
     // ==================== 修饰器 ====================
 
     modifier onlyOwner() {
         require(msg.sender == owner, "AccessControl: only owner");
         _;
     }
 
     modifier onlyActiveUser(address user) {
         require(users[user].active, "AccessControl: user not active");
         _;
     }
 
     // ==================== 构造函数 ====================
 
     constructor(uint256 _windowSeconds, uint256 _maxRequests) {
         owner = msg.sender;
         windowSeconds = _windowSeconds;
         maxRequestsPerWindow = _maxRequests;
 
         users[owner] = UserPermission({
             role: Role.ADMIN,
             maxAccessLevel: DataLevel.TOP_SECRET,
             active: true
         });
         emit UserRegistered(owner, Role.ADMIN, DataLevel.TOP_SECRET);
     }
 
     // ==================== 管理员接口 ====================
 
     function registerUser(address user, Role role, DataLevel maxLevel) external onlyOwner {
         require(role != Role.NONE, "AccessControl: role cannot be NONE");
         require(user != address(0), "AccessControl: invalid address");
         require(!users[user].active, "AccessControl: user already exists");
 
         users[user] = UserPermission({
             role: role,
             maxAccessLevel: maxLevel,
             active: true
         });
         emit UserRegistered(user, role, maxLevel);
     }
 
     function deactivateUser(address user) external onlyOwner {
         require(users[user].active, "AccessControl: user already inactive");
         users[user].active = false;
         emit UserDeactivated(user);
     }
 
     function reactivateUser(address user) external onlyOwner {
         require(!users[user].active, "AccessControl: user already active");
         users[user].active = true;
         emit UserReactivated(user);
     }
 
     function updateRole(address user, Role newRole) external onlyOwner {
         require(users[user].active, "AccessControl: user not active");
         require(newRole != Role.NONE, "AccessControl: role cannot be NONE");
         users[user].role = newRole;
         emit RoleUpdated(user, newRole);
     }
 
     function updateAccessLevel(address user, DataLevel newLevel) external onlyOwner {
         require(users[user].active, "AccessControl: user not active");
         users[user].maxAccessLevel = newLevel;
         emit AccessLevelUpdated(user, newLevel);
     }
 
     function updateRateLimit(uint256 _windowSeconds, uint256 _maxRequests) external onlyOwner {
         windowSeconds = _windowSeconds;
         maxRequestsPerWindow = _maxRequests;
         emit RateLimitUpdated(_windowSeconds, _maxRequests);
     }
 
     // ==================== 核心校验接口 ====================
 
     function checkAccess(address user, DataLevel dataLevel)
         external
         view
         onlyActiveUser(user)
         returns (bool allowed, string memory reason)
     {
         UserPermission memory perm = users[user];
 
         if (perm.role == Role.NONE) {
             return (false, "role not assigned");
         }
 
         if (uint8(dataLevel) > uint8(perm.maxAccessLevel)) {
             return (false, "data level exceeds user clearance");
         }
 
         return (true, "");
     }
 
     function checkRateLimit(address user)
         external
         view
         returns (bool allowed, uint256 remaining)
     {
         RateRecord storage record = rateRecords[user];
         uint256 cutoff = block.timestamp - windowSeconds;
         uint256 count = 0;
 
         for (uint256 i = 0; i < record.timestamps.length; i++) {
             if (record.timestamps[i] >= cutoff) {
                 count++;
             }
         }
 
         if (count >= maxRequestsPerWindow) {
             return (false, 0);
         }
         return (true, maxRequestsPerWindow - count);
     }
 
     /// @notice 记录一次请求时间戳（由链下服务在放行后调用）
     function recordRequest(address user) external onlyOwner {
         RateRecord storage record = rateRecords[user];
         uint256 cutoff = block.timestamp - windowSeconds;
 
         uint256 validIndex = 0;
         for (uint256 i = 0; i < record.timestamps.length; i++) {
             if (record.timestamps[i] >= cutoff) {
                 record.timestamps[validIndex] = record.timestamps[i];
                 validIndex++;
             }
         }
         while (record.timestamps.length > validIndex) {
             record.timestamps.pop();
         }
 
         record.timestamps.push(block.timestamp);
     }
 
     /// @notice 查询用户权限信息
     function getUserInfo(address user)
         external
         view
         returns (Role role, DataLevel maxLevel, bool active)
     {
         UserPermission memory perm = users[user];
         return (perm.role, perm.maxAccessLevel, perm.active);
     }
 }
