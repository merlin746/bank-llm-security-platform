// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @dev 仅声明本测试套件使用的 cheatcode 子集。
interface Vm {
    function prank(address) external;
    function startPrank(address) external;
    function stopPrank() external;
    function warp(uint256) external;
    function expectRevert(bytes calldata) external;
}

/// @title ChainWiseTestBase
/// @notice 链安智御合约测试的最小公共基类。
///
/// @dev 本工程刻意不依赖 forge-std：
///      1. 参赛环境可能无外网，`forge install` 会失败，自包含可保证测试随时可跑；
///      2. 只声明测试真正用到的 cheatcode，避免因 Foundry 版本差异导致编译失败。
///      如需更完整的断言能力，可自行 `forge install foundry-rs/forge-std` 后改用 `Test`。
abstract contract ChainWiseTestBase {
    /// @notice Foundry cheatcode 地址（vm）
    address internal constant VM_ADDRESS = address(uint160(uint256(keccak256("hevm cheat code"))));

    Vm internal constant vm = Vm(VM_ADDRESS);

    // ==================== 断言工具 ====================

    function fail(string memory message) internal pure {
        revert(message);
    }

    function assertTrue(bool condition, string memory message) internal pure {
        if (!condition) revert(message);
    }

    function assertFalse(bool condition, string memory message) internal pure {
        if (condition) revert(message);
    }

    function assertEq(uint256 a, uint256 b, string memory message) internal pure {
        if (a != b) revert(string.concat(message, " (values differ)"));
    }

    function assertEq(address a, address b, string memory message) internal pure {
        if (a != b) revert(string.concat(message, " (addresses differ)"));
    }

    function assertEq(bytes32 a, bytes32 b, string memory message) internal pure {
        if (a != b) revert(string.concat(message, " (bytes32 differ)"));
    }

    function assertEq(string memory a, string memory b, string memory message) internal pure {
        if (keccak256(bytes(a)) != keccak256(bytes(b))) {
            revert(string.concat(message, " (strings differ)"));
        }
    }

    function assertNotEq(bytes32 a, bytes32 b, string memory message) internal pure {
        if (a == b) revert(message);
    }
}
