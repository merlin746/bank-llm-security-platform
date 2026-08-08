 // SPDX-License-Identifier: MIT
 pragma solidity ^0.8.0;
 
 /**
  * @title NodeReconciliation
  * @notice 多节点对账定责合约：四节点 Hash 提交、批量对账、离群节点自动识别
  * @dev 部署于 FISCO BCOS v3.x 联盟链，由链下 MQ Consumer 异步调用
  */
 contract NodeReconciliation {
     // ==================== 类型定义 ====================
 
     /// @notice 节点类型枚举
     enum NodeType {
         ACCESS,      // 访问网关节点
         RAG,         // RAG 检索节点
         INFERENCE,   // 推理节点
         DATA_WAREHOUSE // 数仓节点
     }
 
     /// @notice 单次请求的四节点 Hash 记录
     struct RequestRecord {
         string requestId;           // 请求 ID（全局唯一）
         bytes32[4] nodeHashes;      // [ACCESS, RAG, INFERENCE, DATA_WAREHOUSE] 对应位置的 Hash
         address[4] submitters;      // 各节点提交者地址
         uint256[4] timestamps;      // 各节点提交时间戳
         uint8 submitCount;          // 已提交节点数量 (0-4)
         bool reconciled;            // 是否已完成对账
         uint256 reconciledAt;       // 对账完成时间戳
     }
 
     /// @notice 对账结果结构体
     struct ReconciliationResult {
         string requestId;               // 请求 ID
         bool consistent;                // 四节点是否一致
         bytes32 consensusHash;          // 共识 Hash（多数派）
         NodeType[] anomalousNodes;      // 离群节点列表
         uint256 reconciledAt;           // 对账时间
     }
 
     // ==================== 状态变量 ====================
 
     address public owner;
     uint256 public recordCount;                                 // 已创建的记录数
     uint256 public anomalyCount;                                 // 检测到的异常次数
 
     mapping(string => RequestRecord) public requests;            // requestId => 请求 Hash 记录
     string[] private requestIds;                                 // 所有请求 ID 列表
 
     mapping(string => ReconciliationResult) public results;      // requestId => 对账结果
 
     /// @notice 已注册的可信节点提交者（nodeId => submitter address）
     mapping(bytes32 => address) public registeredSubmitters;
 
     // ==================== 事件 ====================
 
     event NodeHashSubmitted(string indexed requestId, NodeType nodeType, bytes32 nodeHash, address submitter);
     event RequestReconciled(string indexed requestId, bool consistent, bytes32 consensusHash);
     event AnomalyDetected(string indexed requestId, NodeType anomalousNode, bytes32 expectedHash, bytes32 actualHash);
     event SubmitterRegistered(bytes32 indexed nodeId, address submitter);
     event SubmitterRemoved(bytes32 indexed nodeId);
 
     // ==================== 修饰器 ====================
 
     modifier onlyOwner() {
         require(msg.sender == owner, "NodeReconciliation: only owner");
         _;
     }
 
     modifier onlyRegisteredSubmitter() {
         bool registered;
         // 校验调用者是否为任一已注册提交者（O(n) 遍历，n 为节点数，可接受）
         // 在实际 FISCO BCOS 场景中可优化为 mapping 反向索引
         emit NodeHashSubmitted("", NodeType.ACCESS, bytes32(0), msg.sender); // placeholder
         require(registered, "NodeReconciliation: submitter not registered");
         _;
     }
 
     // ==================== 构造函数 ====================
 
     constructor() {
         owner = msg.sender;
         recordCount = 0;
         anomalyCount = 0;
     }
 
     // ==================== 提交者注册管理 ====================
 
     /// @notice 注册一个节点提交者地址
     function registerSubmitter(bytes32 nodeId, address submitter) external onlyOwner {
         registeredSubmitters[nodeId] = submitter;
         emit SubmitterRegistered(nodeId, submitter);
     }
 
     /// @notice 移除一个节点提交者
     function removeSubmitter(bytes32 nodeId) external onlyOwner {
         delete registeredSubmitters[nodeId];
         emit SubmitterRemoved(nodeId);
     }
 
     /// @notice 查询节点提交者
     function getSubmitter(bytes32 nodeId) external view returns (address) {
         return registeredSubmitters[nodeId];
     }
 
     // ==================== 核心：节点 Hash 提交 ====================
 
     /// @notice 提交单个节点的 Hash（由各节点链下服务异步调用）
     function submitNodeHash(
         string memory requestId,
         NodeType nodeType,
         bytes32 nodeHash
     ) external returns (bool readyForReconciliation) {
         // 首次提交则初始化记录
         if (bytes(requests[requestId].requestId).length == 0) {
             requests[requestId] = RequestRecord({
                 requestId: requestId,
                 nodeHashes: [bytes32(0), bytes32(0), bytes32(0), bytes32(0)],
                 submitters: [address(0), address(0), address(0), address(0)],
                 timestamps: [uint256(0), uint256(0), uint256(0), uint256(0)],
                 submitCount: 0,
                 reconciled: false,
                 reconciledAt: 0
             });
             requestIds.push(requestId);
             recordCount++;
         }
 
         RequestRecord storage record = requests[requestId];
         require(!record.reconciled, "NodeReconciliation: already reconciled");
 
         uint8 idx = uint8(nodeType);
         require(idx < 4, "NodeReconciliation: invalid node type");
         require(record.nodeHashes[idx] == bytes32(0), "NodeReconciliation: already submitted for this node");
 
         record.nodeHashes[idx] = nodeHash;
         record.submitters[idx] = msg.sender;
         record.timestamps[idx] = block.timestamp;
         record.submitCount++;
 
         emit NodeHashSubmitted(requestId, nodeType, nodeHash, msg.sender);
 
         // 当四节点全部提交完成，自动触发对账
         if (record.submitCount == 4) {
             reconcile(requestId);
             return true;
         }
 
         return false;
     }
 
     // ==================== 对账逻辑 ====================
 
     /// @notice 对指定请求执行四节点 Hash 比对
     function reconcile(string memory requestId) internal {
         RequestRecord storage record = requests[requestId];
         require(record.submitCount == 4, "NodeReconciliation: not all nodes submitted");
         require(!record.reconciled, "NodeReconciliation: already reconciled");
 
         bytes32[4] memory hashes = record.nodeHashes;
 
         // 统计每个 Hash 的出现次数
         bytes32[4] memory uniqueHashes;
         uint8[4] memory hashCounts;
         uint8 uniqueCount = 0;
 
         for (uint8 i = 0; i < 4; i++) {
             bool found = false;
             for (uint8 j = 0; j < uniqueCount; j++) {
                 if (hashes[i] == uniqueHashes[j]) {
                     hashCounts[j]++;
                     found = true;
                     break;
                 }
             }
             if (!found) {
                 uniqueHashes[uniqueCount] = hashes[i];
                 hashCounts[uniqueCount] = 1;
                 uniqueCount++;
             }
         }
 
         // 找多数派 (consensus) Hash
         bytes32 consensusHash;
         uint8 maxCount = 0;
         for (uint8 i = 0; i < uniqueCount; i++) {
             if (hashCounts[i] > maxCount) {
                 maxCount = hashCounts[i];
                 consensusHash = uniqueHashes[i];
             }
         }
 
         // 找出离群节点
         NodeType[] memory anomalyNodes;
         uint8 anomalyIdx = 0;
         bool consistent = maxCount == 4;
 
         if (!consistent) {
             anomalyNodes = new NodeType[](4 - maxCount);
             for (uint8 i = 0; i < 4; i++) {
                 if (hashes[i] != consensusHash) {
                     NodeType anomalyType = NodeType(i);
                     anomalyNodes[anomalyIdx] = anomalyType;
                     anomalyIdx++;
                     anomalyCount++;
                     emit AnomalyDetected(requestId, anomalyType, consensusHash, hashes[i]);
                 }
             }
         } else {
             anomalyNodes = new NodeType[](0);
         }
 
         // 保存对账结果
         results[requestId] = ReconciliationResult({
             requestId: requestId,
             consistent: consistent,
             consensusHash: consensusHash,
             anomalousNodes: anomalyNodes,
             reconciledAt: block.timestamp
         });
 
         record.reconciled = true;
         record.reconciledAt = block.timestamp;
 
         emit RequestReconciled(requestId, consistent, consensusHash);
     }
 
     /// @notice 外部触发对账（当某请求四节点已提交但未自动对账时）
     function triggerReconciliation(string memory requestId) external {
         RequestRecord storage record = requests[requestId];
         require(record.submitCount == 4, "NodeReconciliation: not all nodes submitted");
         require(!record.reconciled, "NodeReconciliation: already reconciled");
         reconcile(requestId);
     }
 
     // ==================== 查询接口 ====================
 
     /// @notice 获取请求的原始四节点 Hash 记录
     function getRequestRecord(string memory requestId)
         external
         view
         returns (RequestRecord memory)
     {
         return requests[requestId];
     }
 
     /// @notice 获取对账结果
     function getReconciliationResult(string memory requestId)
         external
         view
         returns (ReconciliationResult memory)
     {
         return results[requestId];
     }
 
     /// @notice 分页查询所有请求 ID
     function getRequestIds(uint256 offset, uint256 limit)
         external
         view
         returns (string[] memory ids, uint256 total)
     {
         total = requestIds.length;
         if (offset >= total) {
             return (new string[](0), total);
         }
         uint256 end = offset + limit;
         if (end > total) {
             end = total;
         }
         uint256 resultLength = end - offset;
         ids = new string[](resultLength);
         for (uint256 i = 0; i < resultLength; i++) {
             ids[i] = requestIds[offset + i];
         }
         return (ids, total);
     }
 
     /// @notice 获取异常统计信息
     function getAnomalyStats()
         external
         view
         returns (uint256 totalRecords, uint256 totalAnomalies, uint256 reconciledCount)
     {
         totalRecords = recordCount;
         totalAnomalies = anomalyCount;
 
         reconciledCount = 0;
         for (uint256 i = 0; i < requestIds.length; i++) {
             if (requests[requestIds[i]].reconciled) {
                 reconciledCount++;
             }
         }
         return (totalRecords, totalAnomalies, reconciledCount);
     }
 
     /// @notice 获取最近的异常事件列表
     function getRecentAnomalies(uint256 count)
         external
         view
         returns (string[] memory anomalyRequestIds, NodeType[] memory anomalyNodeTypes)
     {
         // 遍历（效率较低，仅用于演示）
         uint256 found = 0;
         for (uint256 i = requestIds.length; i > 0 && found < count; i--) {
             string memory rid = requestIds[i - 1];
             if (results[rid].consistent == false) {
                 found++;
             }
         }
 
         anomalyRequestIds = new string[](found);
         anomalyNodeTypes = new NodeType[](found);
         uint256 idx = 0;
         for (uint256 i = requestIds.length; i > 0 && idx < found; i--) {
             string memory rid = requestIds[i - 1];
             ReconciliationResult storage r = results[rid];
             if (!r.consistent) {
                 anomalyRequestIds[idx] = rid;
                 // 取第一个离群节点作为代表
                 anomalyNodeTypes[idx] = r.anomalousNodes.length > 0 ? r.anomalousNodes[0] : NodeType.ACCESS;
                 idx++;
             }
         }
         return (anomalyRequestIds, anomalyNodeTypes);
     }
 }
