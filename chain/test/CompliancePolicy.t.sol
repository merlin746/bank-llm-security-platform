// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {CompliancePolicy} from "../contracts/CompliancePolicy.sol";
import {ChainWiseTestBase} from "./ChainWiseTestBase.sol";

/// @title CompliancePolicyTest
/// @notice 合规策略执行合约测试：策略版本控制、规则管理、高风险操作复核状态机。
///
/// @dev 测试数据统一使用 ASCII 字面量：Solidity 的普通 "..." 字符串不允许非 ASCII
///      字符（需要 unicode"..." 前缀）。合约本身对描述文本不做语义解析，
///      因此用 ASCII 数据即可完整覆盖逻辑分支。
contract CompliancePolicyTest is ChainWiseTestBase {
    CompliancePolicy internal cp;

    address internal constant REVIEWER = address(0x2E71E);
    address internal constant OPERATOR = address(0x0DE1);
    address internal constant OUTSIDER = address(0xBAD);

    bytes32 internal constant RULES_ROOT = keccak256("rules-root-v1");

    function setUp() public {
        cp = new CompliancePolicy();
    }

    // ==================== 规则管理 ====================

    function test_AddRuleAndQueryActiveRuleIds() public {
        cp.addRule("RULE_ADS_001", "advertising", "forbid guaranteed returns", 4);
        cp.addRule("RULE_PRIVACY_001", "privacy", "forbid id card output", 5);

        string[] memory active = cp.getActiveRuleIds();
        assertEq(active.length, 2, "two active rules expected");
        assertEq(active[0], "RULE_ADS_001", "first rule id mismatch");
        assertEq(active[1], "RULE_PRIVACY_001", "second rule id mismatch");
    }

    /// @notice 同名 ruleId 再次 addRule 只做更新，不应重复进入规则列表。
    function test_AddRuleTwiceUpdatesWithoutDuplicating() public {
        cp.addRule("RULE_ADS_001", "advertising", "old description", 2);
        cp.addRule("RULE_ADS_001", "advertising", "new description", 5);

        string[] memory active = cp.getActiveRuleIds();
        assertEq(active.length, 1, "rule should not be duplicated");

        (
            string memory ruleId,
            string memory category,
            string memory desc,
            uint256 severity,
            bool isActive
        ) = cp.rules("RULE_ADS_001");

        assertEq(ruleId, "RULE_ADS_001", "ruleId mismatch");
        assertEq(category, "advertising", "category mismatch");
        assertEq(desc, "new description", "description should be updated");
        assertEq(severity, 5, "severity should be updated");
        assertTrue(isActive, "rule should stay active");
    }

    function test_ToggleRuleExcludesFromActiveList() public {
        cp.addRule("RULE_ADS_001", "advertising", "forbid guaranteed returns", 4);
        cp.addRule("RULE_PRIVACY_001", "privacy", "forbid id card output", 5);

        cp.toggleRule("RULE_ADS_001", false);

        string[] memory active = cp.getActiveRuleIds();
        assertEq(active.length, 1, "disabled rule must be excluded");
        assertEq(active[0], "RULE_PRIVACY_001", "remaining rule mismatch");
    }

    function test_AddRuleRevertsOnEmptyId() public {
        vm.expectRevert(bytes("CompliancePolicy: ruleId required"));
        cp.addRule("", "advertising", "desc", 3);
    }

    function test_AddRuleRevertsOnSeverityOutOfRange() public {
        vm.expectRevert(bytes("CompliancePolicy: severity 1-5"));
        cp.addRule("RULE_X", "advertising", "desc", 6);
    }

    function test_AddRuleAcceptsBoundarySeverity() public {
        cp.addRule("RULE_MIN", "advertising", "desc", 1);
        cp.addRule("RULE_MAX", "advertising", "desc", 5);
        assertEq(cp.getActiveRuleIds().length, 2, "severity 1 and 5 are both valid");
    }

    function test_ToggleUnknownRuleReverts() public {
        vm.expectRevert(bytes("CompliancePolicy: rule not found"));
        cp.toggleRule("RULE_NOPE", false);
    }

    function test_OnlyOwnerCanAddRule() public {
        vm.prank(OUTSIDER);
        vm.expectRevert(bytes("CompliancePolicy: only owner"));
        cp.addRule("RULE_X", "advertising", "desc", 3);
    }

    // ==================== 策略版本控制 ====================

    function test_InitialStateHasNoActiveVersion() public {
        assertEq(cp.currentVersionId(), 0, "no version should be active initially");

        (bool passed, uint256 versionId, ) = cp.checkCompliance("hash");
        assertFalse(passed, "compliance check must fail without an active policy");
        assertEq(versionId, 0, "versionId should be 0");
    }

    function test_ProposeAndEnactVersion() public {
        uint256 effectiveAt = block.timestamp;
        uint256 versionId = cp.proposeVersion("v1 initial policy", RULES_ROOT, effectiveAt);
        assertEq(versionId, 1, "first version id should be 1");

        (
            uint256 id,
            string memory desc,
            bytes32 root,
            uint256 eff,
            address proposer,
            bool enacted
        ) = cp.versions(versionId);

        assertEq(id, 1, "versionId mismatch");
        assertEq(desc, "v1 initial policy", "description mismatch");
        assertEq(root, RULES_ROOT, "rulesRootHash mismatch");
        assertEq(eff, effectiveAt, "effectiveTimestamp mismatch");
        assertEq(proposer, address(this), "proposer mismatch");
        assertFalse(enacted, "should not be enacted before enactVersion");

        cp.enactVersion(versionId);
        assertEq(cp.currentVersionId(), versionId, "currentVersionId should update");
    }

    /// @notice 未到生效时间不得生效（事前合规红线）。
    function test_CannotEnactBeforeEffectiveTimestamp() public {
        uint256 versionId = cp.proposeVersion("v1", RULES_ROOT, block.timestamp + 1 days);

        vm.expectRevert(bytes("CompliancePolicy: not effective yet"));
        cp.enactVersion(versionId);
    }

    function test_CanEnactAfterEffectiveTimestamp() public {
        uint256 versionId = cp.proposeVersion("v1", RULES_ROOT, block.timestamp + 1 hours);

        vm.warp(block.timestamp + 1 hours);
        cp.enactVersion(versionId);
        assertEq(cp.currentVersionId(), versionId, "version should be enacted after its time");
    }

    function test_CannotEnactTwice() public {
        uint256 versionId = cp.proposeVersion("v1", RULES_ROOT, block.timestamp);
        cp.enactVersion(versionId);

        vm.expectRevert(bytes("CompliancePolicy: already enacted"));
        cp.enactVersion(versionId);
    }

    function test_CannotEnactUnknownVersion() public {
        vm.expectRevert(bytes("CompliancePolicy: version not found"));
        cp.enactVersion(999);
    }

    /// @notice 版本升级：v2 生效后 currentVersionId 切到 v2，checkCompliance 返回新版本号。
    function test_VersionUpgradeSwitchesActiveVersion() public {
        uint256 v1 = cp.proposeVersion("v1", RULES_ROOT, block.timestamp);
        cp.enactVersion(v1);

        uint256 v2 = cp.proposeVersion("v2", keccak256("rules-root-v2"), block.timestamp);
        cp.enactVersion(v2);

        assertEq(cp.currentVersionId(), v2, "active version should switch to v2");

        (bool passed, uint256 versionId, string memory message) = cp.checkCompliance("content-hash");
        assertTrue(passed, "compliance check should pass with an active version");
        assertEq(versionId, v2, "checkCompliance should report the new version");
        assertEq(message, "off-chain content check required", "message mismatch");
    }

    function test_OnlyOwnerCanProposeVersion() public {
        vm.prank(OUTSIDER);
        vm.expectRevert(bytes("CompliancePolicy: only owner"));
        cp.proposeVersion("v1", RULES_ROOT, block.timestamp);
    }

    // ==================== 审核人管理 ====================

    function test_OwnerIsDefaultReviewer() public {
        assertTrue(cp.reviewers(address(this)), "deployer should be a reviewer");
    }

    function test_AddAndRemoveReviewer() public {
        cp.addReviewer(REVIEWER);
        assertTrue(cp.reviewers(REVIEWER), "reviewer should be added");

        cp.removeReviewer(REVIEWER);
        assertFalse(cp.reviewers(REVIEWER), "reviewer should be removed");
    }

    function test_AddReviewerTwiceReverts() public {
        cp.addReviewer(REVIEWER);

        vm.expectRevert(bytes("CompliancePolicy: already a reviewer"));
        cp.addReviewer(REVIEWER);
    }

    function test_RemoveNonReviewerReverts() public {
        vm.expectRevert(bytes("CompliancePolicy: not a reviewer"));
        cp.removeReviewer(OUTSIDER);
    }

    // ==================== 高风险操作复核状态机 ====================

    function test_SubmitOperationCreatesPendingRecord() public {
        vm.prank(OPERATOR);
        string memory opId = cp.submitOperation("export high-secret client list", "needed for audit");

        assertEq(opId, "OP_1", "first operation id should be OP_1");

        CompliancePolicy.HighRiskOperation memory op = cp.getOperation(opId);
        assertEq(op.operationId, opId, "operationId mismatch");
        assertEq(op.operator, OPERATOR, "operator mismatch");
        assertEq(op.description, "export high-secret client list", "description mismatch");
        assertEq(op.reason, "needed for audit", "reason mismatch");
        assertEq(
            uint256(op.status),
            uint256(CompliancePolicy.ReviewStatus.PENDING),
            "should be PENDING"
        );
        assertEq(op.reviewer, address(0), "reviewer should be empty before review");
        assertEq(op.reviewedAt, 0, "reviewedAt should be 0 before review");

        string[] memory pending = cp.getPendingOperations();
        assertEq(pending.length, 1, "operation should be in pending list");
        assertEq(pending[0], opId, "pending id mismatch");
    }

    function test_ApproveOperationRemovesFromPending() public {
        vm.prank(OPERATOR);
        string memory opId = cp.submitOperation("export list", "for audit");

        cp.reviewOperation(opId, true, "verified, approved");

        CompliancePolicy.HighRiskOperation memory op = cp.getOperation(opId);
        assertEq(
            uint256(op.status),
            uint256(CompliancePolicy.ReviewStatus.APPROVED),
            "should be APPROVED"
        );
        assertEq(op.reviewer, address(this), "reviewer should be recorded");
        assertEq(op.reviewComment, "verified, approved", "comment mismatch");
        assertTrue(op.reviewedAt > 0, "reviewedAt should be set");

        assertEq(cp.getPendingOperations().length, 0, "approved operation must leave pending list");
    }

    function test_RejectOperation() public {
        vm.prank(OPERATOR);
        string memory opId = cp.submitOperation("export list", "insufficient reason");

        cp.reviewOperation(opId, false, "rejected");

        CompliancePolicy.HighRiskOperation memory op = cp.getOperation(opId);
        assertEq(
            uint256(op.status),
            uint256(CompliancePolicy.ReviewStatus.REJECTED),
            "should be REJECTED"
        );
        assertEq(cp.getPendingOperations().length, 0, "rejected operation must leave pending list");
    }

    function test_CannotReviewNonPendingOperation() public {
        vm.prank(OPERATOR);
        string memory opId = cp.submitOperation("export list", "for audit");
        cp.reviewOperation(opId, true, "approved");

        // 已审核通过的操作再次审核应被拒绝
        vm.expectRevert(bytes("CompliancePolicy: not pending"));
        cp.reviewOperation(opId, false, "revote");
    }

    function test_OnlyReviewerCanReview() public {
        vm.prank(OPERATOR);
        string memory opId = cp.submitOperation("export list", "for audit");

        vm.prank(OUTSIDER);
        vm.expectRevert(bytes("CompliancePolicy: not a reviewer"));
        cp.reviewOperation(opId, true, "approved");
    }

    function test_OperationIdsIncrement() public {
        vm.prank(OPERATOR);
        string memory op1 = cp.submitOperation("operation one", "reason one");

        vm.prank(OPERATOR);
        string memory op2 = cp.submitOperation("operation two", "reason two");

        assertEq(op1, "OP_1", "first id mismatch");
        assertEq(op2, "OP_2", "second id mismatch");
        assertEq(cp.getPendingOperations().length, 2, "both should be pending");
    }

    /// @notice 多个待审核操作共用 swap-pop 移除逻辑，只应移除目标项。
    function test_RemovePendingOnlyRemovesTargetOperation() public {
        vm.prank(OPERATOR);
        string memory op1 = cp.submitOperation("operation one", "reason one");
        vm.prank(OPERATOR);
        string memory op2 = cp.submitOperation("operation two", "reason two");
        vm.prank(OPERATOR);
        string memory op3 = cp.submitOperation("operation three", "reason three");

        // 移除中间项，剩余两项应仍可正常审核
        cp.reviewOperation(op2, true, "approved");

        string[] memory pending = cp.getPendingOperations();
        assertEq(pending.length, 2, "two operations should remain pending");

        // 剩余项仍可被正常审核（验证 pendingOperationIds 未被破坏）
        cp.reviewOperation(op1, false, "rejected");
        cp.reviewOperation(op3, true, "approved");

        assertEq(cp.getPendingOperations().length, 0, "all operations should be reviewed");
        assertEq(
            uint256(cp.getOperation(op1).status),
            uint256(CompliancePolicy.ReviewStatus.REJECTED),
            "op1 should be REJECTED"
        );
        assertEq(
            uint256(cp.getOperation(op3).status),
            uint256(CompliancePolicy.ReviewStatus.APPROVED),
            "op3 should be APPROVED"
        );
    }
}
