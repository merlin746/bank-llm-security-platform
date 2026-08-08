 // SPDX-License-Identifier: MIT
 pragma solidity ^0.8.0;
 
 /**
  * @title CompliancePolicy
  * @notice 合规策略执行合约：策略版本控制、风险规则校验、高风险操作人工复核状态机
  * @dev 部署于 FISCO BCOS v3.x 联盟链
  */
 contract CompliancePolicy {
     // ==================== 类型定义 ====================
 
     /// @notice 策略规则结构体
     struct PolicyRule {
         string ruleId;          // 规则 ID（如 "RULE_ADS_001"）
         string category;        // 规则类别（如 "advertising", "investment", "privacy"）
         string description;     // 规则描述
         uint256 severity;       // 严重级别 1-5
         bool active;            // 是否启用
     }
 
     /// @notice 策略版本结构体
     struct PolicyVersion {
         uint256 versionId;              // 版本号（自增）
         string description;             // 版本描述
         bytes32 rulesRootHash;          // 规则集合的 Merkle Root Hash
         uint256 effectiveTimestamp;     // 生效时间戳
         address proposer;               // 提议人
         bool enacted;                   // 是否已生效
     }
 
     /// @notice 高风险操作审核状态
     enum ReviewStatus {
         NONE,           // 未提交
         PENDING,        // 待审核
         APPROVED,       // 已通过
         REJECTED        // 已驳回
     }
 
     /// @notice 高风险操作记录
     struct HighRiskOperation {
         string operationId;             // 操作 ID
         address operator;               // 操作人
         string description;             // 操作描述
         string reason;                  // 申请理由
         ReviewStatus status;            // 审核状态
         address reviewer;               // 审核人
         string reviewComment;           // 审核意见
         uint256 createdAt;              // 创建时间
         uint256 reviewedAt;             // 审核时间
     }
 
     // ==================== 状态变量 ====================
 
     address public owner;                           // 合约拥有者
     uint256 public currentVersionId;                 // 当前生效的策略版本号
     uint256 public nextVersionId;                    // 下一个版本号
     uint256 public nextOperationId;                  // 下一个操作 ID 序号
 
     mapping(uint256 => PolicyVersion) public versions;               // versionId => 版本
     mapping(string => PolicyRule) public rules;                       // ruleId => 规则（链上存储，JSON 内容由链下解析）
     string[] private ruleIds;                                        // 所有规则 ID 列表
 
     mapping(string => HighRiskOperation) public operations;          // operationId => 审核记录
     string[] private pendingOperationIds;                            // 待审核操作 ID 列表
 
     mapping(address => bool) public reviewers;                       // 授权审核人
 
     // ==================== 事件 ====================
 
     event PolicyVersionProposed(uint256 indexed versionId, string description, bytes32 rulesRootHash, uint256 effectiveTimestamp);
     event PolicyVersionEnacted(uint256 indexed versionId);
     event RuleAdded(string indexed ruleId, string category, uint256 severity);
     event RuleToggled(string indexed ruleId, bool active);
     event OperationSubmitted(string indexed operationId, address indexed operator, string description);
     event OperationReviewed(string indexed operationId, ReviewStatus status, address reviewer);
     event ReviewerAdded(address indexed reviewer);
     event ReviewerRemoved(address indexed reviewer);
 
     // ==================== 修饰器 ====================
 
     modifier onlyOwner() {
         require(msg.sender == owner, "CompliancePolicy: only owner");
         _;
     }
 
     modifier onlyReviewer() {
         require(reviewers[msg.sender], "CompliancePolicy: not a reviewer");
         _;
     }
 
     // ==================== 构造函数 ====================
 
     constructor() {
         owner = msg.sender;
         currentVersionId = 0;
         nextVersionId = 1;
         nextOperationId = 1;
 
         // 默认将部署者设为审核人
         reviewers[owner] = true;
         emit ReviewerAdded(owner);
     }
 
     // ==================== 策略规则管理 ====================
 
     /// @notice 添加或更新一条策略规则
     function addRule(
         string memory ruleId,
         string memory category,
         string memory description,
         uint256 severity
     ) external onlyOwner {
         require(bytes(ruleId).length > 0, "CompliancePolicy: ruleId required");
         require(severity >= 1 && severity <= 5, "CompliancePolicy: severity 1-5");
 
         if (bytes(rules[ruleId].ruleId).length == 0) {
             ruleIds.push(ruleId);
         }
 
         rules[ruleId] = PolicyRule({
             ruleId: ruleId,
             category: category,
             description: description,
             severity: severity,
             active: true
         });
 
         emit RuleAdded(ruleId, category, severity);
     }
 
     /// @notice 启用/禁用某条规则
     function toggleRule(string memory ruleId, bool active) external onlyOwner {
         require(bytes(rules[ruleId].ruleId).length > 0, "CompliancePolicy: rule not found");
         rules[ruleId].active = active;
         emit RuleToggled(ruleId, active);
     }
 
     // ==================== 策略版本管理 ====================
 
     /// @notice 提议新策略版本
     function proposeVersion(
         string memory description,
         bytes32 rulesRootHash,
         uint256 effectiveTimestamp
     ) external onlyOwner returns (uint256) {
         uint256 versionId = nextVersionId;
         nextVersionId++;
 
         versions[versionId] = PolicyVersion({
             versionId: versionId,
             description: description,
             rulesRootHash: rulesRootHash,
             effectiveTimestamp: effectiveTimestamp,
             proposer: msg.sender,
             enacted: false
         });
 
         emit PolicyVersionProposed(versionId, description, rulesRootHash, effectiveTimestamp);
         return versionId;
     }
 
     /// @notice 生效指定版本（可由 owner 或超过生效时间后自动触发）
     function enactVersion(uint256 versionId) external onlyOwner {
         PolicyVersion storage version = versions[versionId];
         require(version.versionId > 0, "CompliancePolicy: version not found");
         require(!version.enacted, "CompliancePolicy: already enacted");
         require(block.timestamp >= version.effectiveTimestamp, "CompliancePolicy: not effective yet");
 
         version.enacted = true;
         currentVersionId = versionId;
         emit PolicyVersionEnacted(versionId);
     }
 
     // ==================== 合规校验接口 ====================
 
     /// @notice 校验某段内容是否违反当前生效策略
     /// @dev 链上做轻量校验（如版本存在性），详细语义合规由链下 AI 模块判定
     function checkCompliance(string memory contentHash)
         external
         view
         returns (bool passed, uint256 versionId, string memory message)
     {
         if (currentVersionId == 0) {
             return (false, 0, "no active policy version");
         }
 
         PolicyVersion storage version = versions[currentVersionId];
         // 链上只校验版本是否生效，具体内容合规由链下引擎根据 rulesRootHash 验证
         return (true, currentVersionId, "off-chain content check required");
     }
 
     /// @notice 获取当前生效版本中所有启用的规则 ID 列表
     function getActiveRuleIds() external view returns (string[] memory) {
         uint256 count = 0;
         for (uint256 i = 0; i < ruleIds.length; i++) {
             if (rules[ruleIds[i]].active) {
                 count++;
             }
         }
 
         string[] memory activeIds = new string[](count);
         uint256 idx = 0;
         for (uint256 i = 0; i < ruleIds.length; i++) {
             if (rules[ruleIds[i]].active) {
                 activeIds[idx] = ruleIds[i];
                 idx++;
             }
         }
         return activeIds;
     }
 
     // ==================== 高风险操作审核状态机 ====================
 
     /// @notice 提交高风险操作审核申请
     function submitOperation(string memory description, string memory reason)
         external
         returns (string memory)
     {
         string memory operationId = string(abi.encodePacked("OP_", uint2str(nextOperationId)));
         nextOperationId++;
 
         operations[operationId] = HighRiskOperation({
             operationId: operationId,
             operator: msg.sender,
             description: description,
             reason: reason,
             status: ReviewStatus.PENDING,
             reviewer: address(0),
             reviewComment: "",
             createdAt: block.timestamp,
             reviewedAt: 0
         });
 
         pendingOperationIds.push(operationId);
 
         emit OperationSubmitted(operationId, msg.sender, description);
         return operationId;
     }
 
     /// @notice 审核高风险操作（通过/驳回）
     function reviewOperation(string memory operationId, bool approved, string memory comment)
         external
         onlyReviewer
     {
         HighRiskOperation storage op = operations[operationId];
         require(op.status == ReviewStatus.PENDING, "CompliancePolicy: not pending");
 
         op.status = approved ? ReviewStatus.APPROVED : ReviewStatus.REJECTED;
         op.reviewer = msg.sender;
         op.reviewComment = comment;
         op.reviewedAt = block.timestamp;
 
         // 从待审核列表中移除
         removePendingOperation(operationId);
 
         emit OperationReviewed(operationId, op.status, msg.sender);
     }
 
     // ==================== 审核人管理 ====================
 
     function addReviewer(address reviewer) external onlyOwner {
         require(!reviewers[reviewer], "CompliancePolicy: already a reviewer");
         reviewers[reviewer] = true;
         emit ReviewerAdded(reviewer);
     }
 
     function removeReviewer(address reviewer) external onlyOwner {
         require(reviewers[reviewer], "CompliancePolicy: not a reviewer");
         reviewers[reviewer] = false;
         emit ReviewerRemoved(reviewer);
     }
 
     // ==================== 查询接口 ====================
 
     function getPendingOperations() external view returns (string[] memory) {
         return pendingOperationIds;
     }
 
     function getOperation(string memory operationId)
         external
         view
         returns (HighRiskOperation memory)
     {
         return operations[operationId];
     }
 
     // ==================== 内部工具函数 ====================
 
     function removePendingOperation(string memory operationId) internal {
         for (uint256 i = 0; i < pendingOperationIds.length; i++) {
             if (keccak256(bytes(pendingOperationIds[i])) == keccak256(bytes(operationId))) {
                 pendingOperationIds[i] = pendingOperationIds[pendingOperationIds.length - 1];
                 pendingOperationIds.pop();
                 break;
             }
         }
     }
 
     /// @notice uint 转 string（Solidity 原生工具函数）
     function uint2str(uint256 _i) internal pure returns (string memory str) {
         if (_i == 0) {
             return "0";
         }
         uint256 j = _i;
         uint256 len;
         while (j != 0) {
             len++;
             j /= 10;
         }
         bytes memory bstr = new bytes(len);
         uint256 k = len;
         while (_i != 0) {
             k--;
             bstr[k] = bytes1(uint8(48 + _i % 10));
             _i /= 10;
         }
         return string(bstr);
     }
 }
