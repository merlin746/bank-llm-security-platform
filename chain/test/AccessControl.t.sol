// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {AccessControl} from "../contracts/AccessControl.sol";
import {ChainWiseTestBase} from "./ChainWiseTestBase.sol";

/// @title AccessControlTest
/// @notice 分级权限准入合约测试：角色校验、密级匹配、滑动窗口限流。
contract AccessControlTest is ChainWiseTestBase {
    AccessControl internal ac;

    address internal constant OPERATOR = address(0x0DE1);
    address internal constant AUDITOR = address(0xA0D17);
    address internal constant OUTSIDER = address(0xBAD);

    uint256 internal constant WINDOW = 60; // 滑动窗口 60 秒
    uint256 internal constant MAX_REQ = 3; // 窗口内最多 3 次

    function setUp() public {
        ac = new AccessControl(WINDOW, MAX_REQ);
    }
    /// @dev checkAccess 返回 (allowed, reason)，测试中只关心 allowed。
    function _allowed(address user, AccessControl.DataLevel level) internal view returns (bool) {
        (bool allowed, ) = ac.checkAccess(user, level);
        return allowed;
    }

    // ==================== 构造函数与注册 ====================

    /// @notice 部署者应自动成为 ADMIN/TOP_SECRET。
    function test_OwnerIsAdminWithTopSecret() public {
        assertEq(ac.owner(), address(this), "owner mismatch");

        bool allowed = _allowed(address(this), AccessControl.DataLevel.TOP_SECRET);
        assertTrue(allowed, "owner should access TOP_SECRET");
    }

    function test_RegisterUserAndCheckAccess() public {
        // 操作员：可访问到 CONFIDENTIAL(2)
        ac.registerUser(OPERATOR, AccessControl.Role.OPERATOR, AccessControl.DataLevel.CONFIDENTIAL);

        bool okPublic = _allowed(OPERATOR, AccessControl.DataLevel.PUBLIC);
        bool okInternal = _allowed(OPERATOR, AccessControl.DataLevel.INTERNAL);
        bool okConfidential = _allowed(OPERATOR, AccessControl.DataLevel.CONFIDENTIAL);
        bool okSecret = _allowed(OPERATOR, AccessControl.DataLevel.SECRET);
        bool okTop = _allowed(OPERATOR, AccessControl.DataLevel.TOP_SECRET);

        assertTrue(okPublic, "PUBLIC should be allowed");
        assertTrue(okInternal, "INTERNAL should be allowed");
        assertTrue(okConfidential, "CONFIDENTIAL equals clearance and should be allowed");
        assertFalse(okSecret, "SECRET exceeds clearance and must be denied");
        assertFalse(okTop, "TOP_SECRET exceeds clearance and must be denied");
    }

    /// @notice 密级为 0 的边界：PUBLIC 对任何已激活用户都应放行。
    function test_PublicLevelAllowedForLowestClearance() public {
        ac.registerUser(AUDITOR, AccessControl.Role.AUDITOR, AccessControl.DataLevel.PUBLIC);
        assertTrue(
            _allowed(AUDITOR, AccessControl.DataLevel.PUBLIC),
            "PUBLIC should be accessible with PUBLIC clearance"
        );
    }

    function test_RegisterUserRevertsOnDuplicate() public {
        ac.registerUser(OPERATOR, AccessControl.Role.OPERATOR, AccessControl.DataLevel.INTERNAL);

        vm.expectRevert(bytes("AccessControl: user already exists"));
        ac.registerUser(OPERATOR, AccessControl.Role.MANAGER, AccessControl.DataLevel.SECRET);
    }

    function test_RegisterUserRevertsOnZeroAddress() public {
        vm.expectRevert(bytes("AccessControl: invalid address"));
        ac.registerUser(address(0), AccessControl.Role.OPERATOR, AccessControl.DataLevel.INTERNAL);
    }

    function test_RegisterUserRevertsOnNoneRole() public {
        vm.expectRevert(bytes("AccessControl: role cannot be NONE"));
        ac.registerUser(OPERATOR, AccessControl.Role.NONE, AccessControl.DataLevel.INTERNAL);
    }

    function test_OnlyOwnerCanRegister() public {
        vm.prank(OUTSIDER);
        vm.expectRevert(bytes("AccessControl: only owner"));
        ac.registerUser(OPERATOR, AccessControl.Role.OPERATOR, AccessControl.DataLevel.INTERNAL);
    }

    // ==================== 停用 / 重新启用 ====================

    /// @notice 停用用户不得再通过访问校验（modifier 直接 revert）。
    function test_DeactivatedUserCannotAccess() public {
        ac.registerUser(OPERATOR, AccessControl.Role.OPERATOR, AccessControl.DataLevel.SECRET);
        ac.deactivateUser(OPERATOR);

        vm.expectRevert(bytes("AccessControl: user not active"));
        ac.checkAccess(OPERATOR, AccessControl.DataLevel.PUBLIC);
    }

    function test_ReactivateRestoresOriginalPermission() public {
        ac.registerUser(OPERATOR, AccessControl.Role.MANAGER, AccessControl.DataLevel.SECRET);
        ac.deactivateUser(OPERATOR);
        ac.reactivateUser(OPERATOR);

        (AccessControl.Role role, AccessControl.DataLevel level, bool active) =
            ac.getUserInfo(OPERATOR);

        assertEq(uint256(role), uint256(AccessControl.Role.MANAGER), "role should be preserved");
        assertEq(uint256(level), uint256(AccessControl.DataLevel.SECRET), "level should be preserved");
        assertTrue(active, "user should be active again");
    }

    function test_DeactivateInactiveUserReverts() public {
        ac.registerUser(OPERATOR, AccessControl.Role.OPERATOR, AccessControl.DataLevel.INTERNAL);
        ac.deactivateUser(OPERATOR);

        vm.expectRevert(bytes("AccessControl: user already inactive"));
        ac.deactivateUser(OPERATOR);
    }

    function test_ReactivateActiveUserReverts() public {
        ac.registerUser(OPERATOR, AccessControl.Role.OPERATOR, AccessControl.DataLevel.INTERNAL);

        vm.expectRevert(bytes("AccessControl: user already active"));
        ac.reactivateUser(OPERATOR);
    }

    // ==================== 权限变更 ====================

    function test_UpdateRoleAndAccessLevel() public {
        ac.registerUser(OPERATOR, AccessControl.Role.OPERATOR, AccessControl.DataLevel.INTERNAL);

        // 提权
        ac.updateRole(OPERATOR, AccessControl.Role.MANAGER);
        ac.updateAccessLevel(OPERATOR, AccessControl.DataLevel.TOP_SECRET);

        (AccessControl.Role role, AccessControl.DataLevel level, ) = ac.getUserInfo(OPERATOR);
        assertEq(uint256(role), uint256(AccessControl.Role.MANAGER), "role not updated");
        assertEq(uint256(level), uint256(AccessControl.DataLevel.TOP_SECRET), "level not updated");

        assertTrue(
            _allowed(OPERATOR, AccessControl.DataLevel.TOP_SECRET),
            "elevated user should access TOP_SECRET"
        );
    }

    /// @notice 降权后原先可访问的密级应立即被拒绝。
    function test_DowngradeTakesEffectImmediately() public {
        ac.registerUser(OPERATOR, AccessControl.Role.MANAGER, AccessControl.DataLevel.SECRET);
        assertTrue(_allowed(OPERATOR, AccessControl.DataLevel.SECRET), "precondition");

        ac.updateAccessLevel(OPERATOR, AccessControl.DataLevel.PUBLIC);

        assertFalse(
            _allowed(OPERATOR, AccessControl.DataLevel.INTERNAL),
            "downgrade must take effect immediately"
        );
    }

    function test_UpdateRoleRevertsForInactiveUser() public {
        ac.registerUser(OPERATOR, AccessControl.Role.OPERATOR, AccessControl.DataLevel.INTERNAL);
        ac.deactivateUser(OPERATOR);

        vm.expectRevert(bytes("AccessControl: user not active"));
        ac.updateRole(OPERATOR, AccessControl.Role.MANAGER);
    }

    function test_UpdateRoleRevertsOnNoneRole() public {
        ac.registerUser(OPERATOR, AccessControl.Role.OPERATOR, AccessControl.DataLevel.INTERNAL);

        vm.expectRevert(bytes("AccessControl: role cannot be NONE"));
        ac.updateRole(OPERATOR, AccessControl.Role.NONE);
    }

    // ==================== 滑动窗口限流 ====================

    function test_RateLimitAllowsUpToMax() public {
        (bool allowed, uint256 remaining) = ac.checkRateLimit(OPERATOR);
        assertTrue(allowed, "fresh user should be allowed");
        assertEq(remaining, MAX_REQ, "remaining should equal max");

        ac.recordRequest(OPERATOR);

        (allowed, remaining) = ac.checkRateLimit(OPERATOR);
        assertTrue(allowed, "should still be allowed after 1 request");
        assertEq(remaining, MAX_REQ - 1, "remaining should decrease");
    }

    function test_RateLimitBlocksAtMax() public {
        ac.recordRequest(OPERATOR);
        ac.recordRequest(OPERATOR);
        ac.recordRequest(OPERATOR);

        (bool allowed, uint256 remaining) = ac.checkRateLimit(OPERATOR);
        assertFalse(allowed, "should be blocked at max requests");
        assertEq(remaining, 0, "remaining should be 0");
    }

    /// @notice 窗口滑过后，过期时间戳被清理，配额恢复。
    function test_RateLimitWindowSlides() public {
        ac.recordRequest(OPERATOR);
        ac.recordRequest(OPERATOR);
        ac.recordRequest(OPERATOR);

        (bool blocked, ) = ac.checkRateLimit(OPERATOR);
        assertFalse(blocked, "precondition: blocked at max");

        // 时间前进超过窗口长度
        vm.warp(block.timestamp + WINDOW + 1);

        (bool allowed, uint256 remaining) = ac.checkRateLimit(OPERATOR);
        assertTrue(allowed, "quota should recover after window slides");
        assertEq(remaining, MAX_REQ, "all timestamps should have expired");
    }

    /// @notice 部分过期：窗口内应只统计未过期的记录。
    function test_RateLimitCountsOnlyNonExpiredRecords() public {
        ac.recordRequest(OPERATOR);
        ac.recordRequest(OPERATOR);

        // 前进半个窗口后再记一次
        vm.warp(block.timestamp + WINDOW / 2);
        ac.recordRequest(OPERATOR);

        (bool allowed, uint256 remaining) = ac.checkRateLimit(OPERATOR);
        // 三条记录都还在窗口内，已达上限 -> 应被拒
        assertFalse(allowed, "three records inside the window reach the limit");
        assertEq(remaining, 0, "remaining should be 0 at the limit");

        // 再过半个窗口：最早两条过期，只剩 1 条
        vm.warp(block.timestamp + WINDOW / 2 + 1);
        (allowed, remaining) = ac.checkRateLimit(OPERATOR);
        assertTrue(allowed, "expired records should free quota");
        assertEq(remaining, MAX_REQ - 1, "only one record should remain in window");
    }

    /// @notice recordRequest 会压缩清理过期记录，使数组长度不无限增长。
    function test_RecordRequestPrunesExpiredEntries() public {
        ac.recordRequest(OPERATOR);
        ac.recordRequest(OPERATOR);

        vm.warp(block.timestamp + WINDOW + 1);

        // 触发一次清理
        ac.recordRequest(OPERATOR);

        // 清理后窗口内只有 1 条 -> remaining = MAX-1
        (, uint256 remaining) = ac.checkRateLimit(OPERATOR);
        assertEq(remaining, MAX_REQ - 1, "expired entries should be pruned");
    }

    function test_OnlyOwnerCanRecordRequest() public {
        vm.prank(OUTSIDER);
        vm.expectRevert(bytes("AccessControl: only owner"));
        ac.recordRequest(OPERATOR);
    }

    function test_UpdateRateLimit() public {
        ac.updateRateLimit(120, 10);

        assertEq(ac.windowSeconds(), 120, "windowSeconds not updated");
        assertEq(ac.maxRequestsPerWindow(), 10, "maxRequestsPerWindow not updated");

        ac.recordRequest(OPERATOR);
        (, uint256 remaining) = ac.checkRateLimit(OPERATOR);
        assertEq(remaining, 9, "new limit should apply");
    }

    /// @notice 回归测试：链上时间戳小于窗口长度时，限流逻辑不得下溢 revert。
    /// @dev 修复前 `block.timestamp - windowSeconds` 在 0.8 语义下会 panic(0x11)，
    ///      导致 checkRateLimit / recordRequest 在本地链、测试链或大窗口配置下
    ///      完全不可用。修复后窗口起点退化为 0。
    function test_RateLimitDoesNotUnderflowWhenTimestampBelowWindow() public {
        // 构造一个窗口远大于当前链上时间的场景
        AccessControl bigWindow = new AccessControl(1_000_000, 5);

        assertTrue(block.timestamp < 1_000_000, "precondition: chain time is below the window");

        (bool allowed, uint256 remaining) = bigWindow.checkRateLimit(OPERATOR);
        assertTrue(allowed, "must not revert and should allow");
        assertEq(remaining, 5, "no history -> full quota");

        // recordRequest 同样不得 revert
        bigWindow.recordRequest(OPERATOR);

        (, remaining) = bigWindow.checkRateLimit(OPERATOR);
        assertEq(remaining, 4, "recorded request should consume one slot");
    }

    function test_GetUserInfoForUnknownUser() public {
        (AccessControl.Role role, AccessControl.DataLevel level, bool active) =
            ac.getUserInfo(OUTSIDER);

        assertEq(uint256(role), uint256(AccessControl.Role.NONE), "unknown user has NONE role");
        assertEq(uint256(level), uint256(AccessControl.DataLevel.PUBLIC), "default level is PUBLIC");
        assertFalse(active, "unknown user should be inactive");
    }
}
