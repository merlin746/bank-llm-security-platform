// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {NodeReconciliation} from "../contracts/NodeReconciliation.sol";
import {ChainWiseTestBase} from "./ChainWiseTestBase.sol";

/// @title NodeReconciliationTest
/// @notice 多节点对账定责合约测试，重点覆盖提交者身份校验与零 Hash 提交两处缺陷。
///
/// @dev 说明：Solidity 为 public 状态变量自动生成的 getter 会**省略**结构体中的
///      动态类型成员（string / 动态数组）。因此：
///        - `requests(id)` 返回 (nodeHashes, submitters, timestamps, submitted,
///          submitCount, reconciled, reconciledAt)，不含 string requestId；
///        - `results(id)` 返回 (consistent, consensusHash, reconciledAt)，
///          不含 NodeType[] anomalousNodes。
///      需要完整结构体（含离群节点列表）时使用 `getReconciliationResult`。
contract NodeReconciliationTest is ChainWiseTestBase {
    NodeReconciliation internal recon;

    address internal constant SUB_ACCESS = address(0xA11CE);
    address internal constant SUB_RAG = address(0xB0B);
    address internal constant SUB_INFERENCE = address(0xCA401);
    address internal constant SUB_WAREHOUSE = address(0xDA7A);

    string internal constant REQ = "REQ-20260826-0001";

    bytes32 internal constant HASH_ACCESS = keccak256("access-payload");
    bytes32 internal constant HASH_RAG = keccak256("rag-payload");
    bytes32 internal constant HASH_INFERENCE = keccak256("inference-payload");
    bytes32 internal constant HASH_WAREHOUSE = keccak256("warehouse-payload");

    function setUp() public {
        recon = new NodeReconciliation();
    }

    // ==================== 测试辅助 ====================

    /// @dev 注册四个业务节点提交者（部署者由构造函数自动注册为初始可信提交者）。
    function _registerFourNodes() internal {
        recon.registerSubmitter(bytes32("ACCESS"), SUB_ACCESS);
        recon.registerSubmitter(bytes32("RAG"), SUB_RAG);
        recon.registerSubmitter(bytes32("INFERENCE"), SUB_INFERENCE);
        recon.registerSubmitter(bytes32("DATA_WAREHOUSE"), SUB_WAREHOUSE);
    }

    /// @dev 依次用四个提交者身份提交四节点 Hash。
    function _submitAll(
        string memory requestId,
        bytes32 h0,
        bytes32 h1,
        bytes32 h2,
        bytes32 h3
    ) internal {
        vm.prank(SUB_ACCESS);
        recon.submitNodeHash(requestId, NodeReconciliation.NodeType.ACCESS, h0);

        vm.prank(SUB_RAG);
        recon.submitNodeHash(requestId, NodeReconciliation.NodeType.RAG, h1);

        vm.prank(SUB_INFERENCE);
        recon.submitNodeHash(requestId, NodeReconciliation.NodeType.INFERENCE, h2);

        vm.prank(SUB_WAREHOUSE);
        recon.submitNodeHash(requestId, NodeReconciliation.NodeType.DATA_WAREHOUSE, h3);
    }

    /// @dev 读取请求记录的提交数量与对账标志。
    ///      统一走 `getRequestRecord`（返回完整结构体），避免依赖 public 状态变量
    ///      自动 getter 的组件展开规则（编译器会省略 string 成员）。
    function _recordState(string memory requestId)
        internal
        view
        returns (uint8 submitCount, bool reconciled)
    {
        NodeReconciliation.RequestRecord memory rec = recon.getRequestRecord(requestId);
        return (rec.submitCount, rec.reconciled);
    }

    /// @dev 读取对账结果的核心字段（完整结构体，含离群节点列表）。
    function _result(string memory requestId)
        internal
        view
        returns (NodeReconciliation.ReconciliationResult memory)
    {
        return recon.getReconciliationResult(requestId);
    }

    // ==================== 缺陷回归测试 ====================

    /// @notice 回归测试（缺陷一）：未注册地址不得提交节点 Hash。
    /// @dev 修复前 onlyRegisteredSubmitter 中 registered 恒为 false，且该修饰器
    ///      未被任何函数使用，导致任意地址都能提交，链上存证完全失去可信性。
    function test_UnregisteredSubmitterIsRejected() public {
        address attacker = address(0xBAD);

        vm.prank(attacker);
        vm.expectRevert(bytes("NodeReconciliation: submitter not registered"));
        recon.submitNodeHash(REQ, NodeReconciliation.NodeType.ACCESS, HASH_ACCESS);

        // 被拒绝的提交不得产生任何记录
        assertEq(recon.recordCount(), 0, "rejected submit must not create a record");
    }

    /// @notice 回归测试（缺陷一）：注册后即可正常提交，且提交者被如实记录。
    function test_RegisteredSubmitterIsAccepted() public {
        recon.registerSubmitter(bytes32("ACCESS"), SUB_ACCESS);

        vm.prank(SUB_ACCESS);
        bool ready = recon.submitNodeHash(REQ, NodeReconciliation.NodeType.ACCESS, HASH_ACCESS);

        assertFalse(ready, "should not be ready after a single submit");

        (uint8 submitCount, ) = _recordState(REQ);
        assertEq(submitCount, 1, "submitCount should be 1");

        // 校验 submitters[ACCESS] 被记录（经完整结构体读取）
        NodeReconciliation.RequestRecord memory rec = recon.getRequestRecord(REQ);
        assertEq(rec.submitters[0], SUB_ACCESS, "submitter should be recorded");
    }

    /// @notice 回归测试（缺陷一）：owner 作为初始可信提交者开箱可用。
    function test_OwnerIsBootstrapSubmitter() public {
        assertTrue(
            recon.isRegisteredSubmitter(address(this)),
            "deployer should be registered as bootstrap submitter"
        );

        bool ready = recon.submitNodeHash(REQ, NodeReconciliation.NodeType.ACCESS, HASH_ACCESS);
        assertFalse(ready, "single submit should not complete reconciliation");
    }

    /// @notice 回归测试（缺陷二）：bytes32(0) 是合法 Hash，必须能提交。
    /// @dev 修复前以 `nodeHashes[idx] == bytes32(0)` 判断重复提交，会把合法提交
    ///      误判为「尚未提交」，使 recordCount/submitCount 与真实状态脱节。
    function test_ZeroHashIsAcceptedAsValidSubmission() public {
        _registerFourNodes();

        // 四节点全部提交零 Hash：应当合法，且完全一致 -> 不产生异常
        _submitAll(REQ, bytes32(0), bytes32(0), bytes32(0), bytes32(0));

        (uint8 submitCount, bool reconciled) = _recordState(REQ);
        assertEq(submitCount, 4, "four zero-hash submits should all be counted");
        assertTrue(reconciled, "zero-hash quorum should still reconcile");

        NodeReconciliation.ReconciliationResult memory res = _result(REQ);
        bool consistent = res.consistent;
        bytes32 consensusHash = res.consensusHash;
        assertTrue(consistent, "identical zero hashes are consistent");
        assertEq(consensusHash, bytes32(0), "consensus hash should be zero");
        assertEq(recon.anomalyCount(), 0, "no anomaly expected for identical hashes");
    }

    /// @notice 回归测试（缺陷二）：即使首次提交的 Hash 为 0，同节点重复提交仍须被拒。
    function test_DuplicateSubmitRejectedEvenWithZeroHash() public {
        recon.registerSubmitter(bytes32("ACCESS"), SUB_ACCESS);

        vm.prank(SUB_ACCESS);
        recon.submitNodeHash(REQ, NodeReconciliation.NodeType.ACCESS, bytes32(0));

        vm.prank(SUB_ACCESS);
        vm.expectRevert(bytes("NodeReconciliation: already submitted for this node"));
        recon.submitNodeHash(REQ, NodeReconciliation.NodeType.ACCESS, HASH_ACCESS);
    }

    // ==================== 提交者注册管理 ====================

    function test_OnlyOwnerCanRegisterSubmitter() public {
        vm.prank(address(0xBAD));
        vm.expectRevert(bytes("NodeReconciliation: only owner"));
        recon.registerSubmitter(bytes32("ACCESS"), SUB_ACCESS);
    }

    function test_CannotRegisterSameNodeIdTwice() public {
        recon.registerSubmitter(bytes32("ACCESS"), SUB_ACCESS);

        vm.expectRevert(bytes("NodeReconciliation: nodeId already registered"));
        recon.registerSubmitter(bytes32("ACCESS"), address(0xE1E));
    }

    function test_CannotRegisterSameAddressTwice() public {
        recon.registerSubmitter(bytes32("ACCESS"), SUB_ACCESS);

        vm.expectRevert(bytes("NodeReconciliation: submitter already registered"));
        recon.registerSubmitter(bytes32("RAG"), SUB_ACCESS);
    }

    function test_CannotRegisterZeroAddress() public {
        vm.expectRevert(bytes("NodeReconciliation: invalid submitter"));
        recon.registerSubmitter(bytes32("ACCESS"), address(0));
    }

    /// @notice 移除后反向索引必须同步清理，否则会残留「已注册」假状态。
    function test_RemoveSubmitterClearsReverseIndex() public {
        recon.registerSubmitter(bytes32("ACCESS"), SUB_ACCESS);
        assertTrue(recon.isRegisteredSubmitter(SUB_ACCESS), "should be registered");

        recon.removeSubmitter(bytes32("ACCESS"));

        assertFalse(recon.isRegisteredSubmitter(SUB_ACCESS), "reverse index must be cleared");
        assertEq(recon.getSubmitter(bytes32("ACCESS")), address(0), "forward index must be cleared");

        // 移除后不能再提交
        vm.prank(SUB_ACCESS);
        vm.expectRevert(bytes("NodeReconciliation: submitter not registered"));
        recon.submitNodeHash(REQ, NodeReconciliation.NodeType.ACCESS, HASH_ACCESS);
    }

    /// @notice 改绑流程：必须先移除再注册，且改绑后原地址失效、新地址生效。
    function test_RebindSubmitterAfterRemoval() public {
        recon.registerSubmitter(bytes32("ACCESS"), SUB_ACCESS);
        recon.removeSubmitter(bytes32("ACCESS"));

        address newSubmitter = address(0xE1E);
        recon.registerSubmitter(bytes32("ACCESS"), newSubmitter);

        vm.prank(SUB_ACCESS);
        vm.expectRevert(bytes("NodeReconciliation: submitter not registered"));
        recon.submitNodeHash(REQ, NodeReconciliation.NodeType.ACCESS, HASH_ACCESS);

        vm.prank(newSubmitter);
        recon.submitNodeHash(REQ, NodeReconciliation.NodeType.ACCESS, HASH_ACCESS);
        assertEq(recon.recordCount(), 1, "new submitter should be able to submit");
    }

    function test_RemoveUnregisteredNodeReverts() public {
        vm.expectRevert(bytes("NodeReconciliation: nodeId not registered"));
        recon.removeSubmitter(bytes32("NOPE"));
    }

    // ==================== 对账核心逻辑 ====================

    /// @notice 四节点 Hash 完全一致：consistent，无离群节点。
    function test_ReconcileConsistent() public {
        _registerFourNodes();
        _submitAll(REQ, HASH_ACCESS, HASH_ACCESS, HASH_ACCESS, HASH_ACCESS);

        NodeReconciliation.ReconciliationResult memory res = _result(REQ);
        bool consistent = res.consistent;
        bytes32 consensusHash = res.consensusHash;
        assertTrue(consistent, "identical hashes should be consistent");
        assertEq(consensusHash, HASH_ACCESS, "consensus hash mismatch");
        assertEq(recon.anomalyCount(), 0, "anomalyCount should stay 0");

        NodeReconciliation.ReconciliationResult memory r = _result(REQ);
        assertEq(r.anomalousNodes.length, 0, "no anomalous node expected");

        (uint256 totalRecords, uint256 totalAnomalies, uint256 reconciledCount) =
            recon.getAnomalyStats();
        assertEq(totalRecords, 1, "totalRecords should be 1");
        assertEq(totalAnomalies, 0, "totalAnomalies should be 0");
        assertEq(reconciledCount, 1, "reconciledCount should be 1");
    }

    /// @notice 三对一：多数派为 consensus，落单节点被判为离群。
    function test_ReconcileDetectsOutlierNode() public {
        _registerFourNodes();
        // 仅 RAG 节点 Hash 不同
        _submitAll(REQ, HASH_ACCESS, HASH_RAG, HASH_ACCESS, HASH_ACCESS);

        NodeReconciliation.ReconciliationResult memory res = _result(REQ);
        bool consistent = res.consistent;
        bytes32 consensusHash = res.consensusHash;
        assertFalse(consistent, "should be inconsistent");
        assertEq(consensusHash, HASH_ACCESS, "majority hash should win");
        assertEq(recon.anomalyCount(), 1, "anomalyCount should be 1");

        NodeReconciliation.ReconciliationResult memory r = _result(REQ);
        assertEq(r.anomalousNodes.length, 1, "exactly one anomalous node");
        assertEq(
            uint256(r.anomalousNodes[0]),
            uint256(NodeReconciliation.NodeType.RAG),
            "RAG should be the anomalous node"
        );
    }

    /// @notice 二对二平局：无绝对多数，判为不一致，并把偏离参考 Hash 的两个节点标为离群。
    /// @dev 合约统计众数时按「先出现者优先」（`hashCounts[i] > maxCount` 为严格大于），
    ///      因此 2-2 平局下参考 Hash 取先提交的那一组，另两个节点被判为离群。
    ///      该行为是确定性的：判定结果只取决于提交顺序，不依赖其他因素。
    ///      注意此时 anomalousNodes.length == 2 而非 4 —— 只标记真正偏离参考值的节点。
    function test_ReconcileTwoTwoSplitIsInconsistent() public {
        _registerFourNodes();
        _submitAll(REQ, HASH_ACCESS, HASH_ACCESS, HASH_INFERENCE, HASH_INFERENCE);

        NodeReconciliation.ReconciliationResult memory res = _result(REQ);
        assertFalse(res.consistent, "2-2 split has no majority and must be inconsistent");
        assertEq(res.consensusHash, HASH_ACCESS, "first-seen hash becomes the reference");

        assertEq(res.anomalousNodes.length, 2, "the two minority nodes should be flagged");
        assertEq(
            uint256(res.anomalousNodes[0]),
            uint256(NodeReconciliation.NodeType.INFERENCE),
            "INFERENCE diverges from the reference hash"
        );
        assertEq(
            uint256(res.anomalousNodes[1]),
            uint256(NodeReconciliation.NodeType.DATA_WAREHOUSE),
            "DATA_WAREHOUSE diverges from the reference hash"
        );
        assertEq(recon.anomalyCount(), 2, "anomalyCount counts each divergent node");
    }

    function test_SubmitReturnsTrueOnFourthSubmission() public {
        _registerFourNodes();

        vm.prank(SUB_ACCESS);
        recon.submitNodeHash(REQ, NodeReconciliation.NodeType.ACCESS, HASH_ACCESS);

        vm.prank(SUB_RAG);
        recon.submitNodeHash(REQ, NodeReconciliation.NodeType.RAG, HASH_RAG);

        vm.prank(SUB_INFERENCE);
        recon.submitNodeHash(REQ, NodeReconciliation.NodeType.INFERENCE, HASH_INFERENCE);

        vm.prank(SUB_WAREHOUSE);
        bool ready = recon.submitNodeHash(
            REQ,
            NodeReconciliation.NodeType.DATA_WAREHOUSE,
            HASH_WAREHOUSE
        );

        assertTrue(ready, "fourth submit should report readyForReconciliation");
    }

    function test_CannotSubmitAfterReconciled() public {
        _registerFourNodes();
        _submitAll(REQ, HASH_ACCESS, HASH_ACCESS, HASH_ACCESS, HASH_ACCESS);

        vm.prank(SUB_ACCESS);
        vm.expectRevert(bytes("NodeReconciliation: already reconciled"));
        recon.submitNodeHash(REQ, NodeReconciliation.NodeType.ACCESS, HASH_RAG);
    }

    function test_TriggerReconciliationRequiresFourSubmits() public {
        _registerFourNodes();

        vm.prank(SUB_ACCESS);
        recon.submitNodeHash(REQ, NodeReconciliation.NodeType.ACCESS, HASH_ACCESS);

        vm.expectRevert(bytes("NodeReconciliation: not all nodes submitted"));
        recon.triggerReconciliation(REQ);
    }

    /// @notice 手工触发对账：第四次提交已自动完成对账时，再次触发应被拒绝。
    function test_TriggerReconciliationAfterAutoReconcileReverts() public {
        _registerFourNodes();
        _submitAll(REQ, HASH_ACCESS, HASH_ACCESS, HASH_ACCESS, HASH_ACCESS);

        vm.expectRevert(bytes("NodeReconciliation: already reconciled"));
        recon.triggerReconciliation(REQ);
    }

    // ==================== 查询接口 ====================

    function test_RecordCountIncrementsOncePerRequest() public {
        _registerFourNodes();

        vm.prank(SUB_ACCESS);
        recon.submitNodeHash(REQ, NodeReconciliation.NodeType.ACCESS, HASH_ACCESS);
        vm.prank(SUB_RAG);
        recon.submitNodeHash(REQ, NodeReconciliation.NodeType.RAG, HASH_RAG);
        assertEq(recon.recordCount(), 1, "multiple submits on one request still count as one record");

        vm.prank(SUB_ACCESS);
        recon.submitNodeHash("REQ-20260826-0002", NodeReconciliation.NodeType.ACCESS, HASH_ACCESS);
        assertEq(recon.recordCount(), 2, "a new request id creates a new record");
    }

    function test_GetRequestIdsPagination() public {
        _registerFourNodes();

        for (uint256 i = 1; i <= 5; i++) {
            string memory rid = string.concat("REQ-", _toString(i));
            vm.prank(SUB_ACCESS);
            recon.submitNodeHash(rid, NodeReconciliation.NodeType.ACCESS, HASH_ACCESS);
        }

        (string[] memory ids, uint256 total) = recon.getRequestIds(0, 3);
        assertEq(total, 5, "total should be 5");
        assertEq(ids.length, 3, "first page should return 3 ids");
        assertEq(ids[0], "REQ-1", "first id mismatch");

        (string[] memory page2, ) = recon.getRequestIds(3, 3);
        assertEq(page2.length, 2, "second page should return remaining 2 ids");

        (string[] memory empty, ) = recon.getRequestIds(10, 3);
        assertEq(empty.length, 0, "out-of-range offset returns empty page");
    }

    /// @notice 异常列表：只回传不一致的请求，并给出代表性离群节点。
    function test_GetRecentAnomaliesReturnsOnlyInconsistentRequests() public {
        _registerFourNodes();

        // REQ-OK：四节点一致
        _submitAll("REQ-OK", HASH_ACCESS, HASH_ACCESS, HASH_ACCESS, HASH_ACCESS);
        // REQ-BAD：RAG 节点离群
        _submitAll("REQ-BAD", HASH_ACCESS, HASH_RAG, HASH_ACCESS, HASH_ACCESS);

        (string[] memory anomalyIds, NodeReconciliation.NodeType[] memory nodeTypes) =
            recon.getRecentAnomalies(10);

        assertEq(anomalyIds.length, 1, "only the inconsistent request should be reported");
        assertEq(anomalyIds[0], "REQ-BAD", "wrong request reported as anomaly");
        assertEq(uint256(nodeTypes[0]), uint256(NodeReconciliation.NodeType.RAG), "wrong node type");
    }

    function _toString(uint256 value) internal pure returns (string memory) {
        if (value == 0) return "0";
        uint256 temp = value;
        uint256 digits;
        while (temp != 0) {
            digits++;
            temp /= 10;
        }
        bytes memory buffer = new bytes(digits);
        while (value != 0) {
            digits -= 1;
            buffer[digits] = bytes1(uint8(48 + (value % 10)));
            value /= 10;
        }
        return string(buffer);
    }
}
