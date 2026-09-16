package model

import (
	"encoding/json"
	"strings"
	"testing"
)

// ==================== 统一响应结构 ====================

func TestSuccessResponse(t *testing.T) {
	resp := Success(map[string]int{"total": 7})

	if resp.Code != 0 {
		t.Errorf("Success code = %d, want 0", resp.Code)
	}
	if resp.Message != "success" {
		t.Errorf("Success message = %q, want success", resp.Message)
	}
	if resp.Msg != "ok" {
		t.Errorf("Success msg = %q, want ok", resp.Msg)
	}
	if resp.Data == nil {
		t.Error("Success should carry data")
	}
}

func TestErrorResponse(t *testing.T) {
	resp := Error(404, "user not found")

	if resp.Code != 404 {
		t.Errorf("Error code = %d, want 404", resp.Code)
	}
	if resp.Message != "user not found" {
		t.Errorf("Error message = %q", resp.Message)
	}
	if resp.Msg != "user not found" {
		t.Errorf("Error msg = %q, want it to mirror message", resp.Msg)
	}
	if resp.Data != nil {
		t.Errorf("Error should not carry data, got %v", resp.Data)
	}
}

// TestResponseJSONShape 锁定前端消费的字段名（code/msg/data）。
func TestResponseJSONShape(t *testing.T) {
	raw, err := json.Marshal(Success([]string{"a"}))
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	for _, key := range []string{"code", "message", "msg", "data"} {
		if _, ok := decoded[key]; !ok {
			t.Errorf("response JSON missing key %q: %s", key, string(raw))
		}
	}
}

// TestErrorResponseOmitsData data 为空时应被 omitempty 省略。
func TestErrorResponseOmitsData(t *testing.T) {
	raw, err := json.Marshal(Error(500, "boom"))
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if strings.Contains(string(raw), `"data"`) {
		t.Errorf("data should be omitted for errors, got %s", string(raw))
	}
}

// ==================== 枚举 ====================

func TestNodeTypeString(t *testing.T) {
	cases := []struct {
		nodeType NodeType
		want     string
	}{
		{NodeAccess, "ACCESS"},
		{NodeRAG, "RAG"},
		{NodeInference, "INFERENCE"},
		{NodeDataWarehouse, "DATA_WAREHOUSE"},
		{NodeType(99), "UNKNOWN"},
	}

	for _, c := range cases {
		if got := c.nodeType.String(); got != c.want {
			t.Errorf("NodeType(%d).String() = %q, want %q", c.nodeType, got, c.want)
		}
	}
}

// TestNodeTypeOrdinalsMatchContract 枚举序号必须与 Solidity 合约一致，
// 否则链上链下对 NodeType 的解读会错位。
func TestNodeTypeOrdinalsMatchContract(t *testing.T) {
	if NodeAccess != 0 || NodeRAG != 1 || NodeInference != 2 || NodeDataWarehouse != 3 {
		t.Fatalf("NodeType ordinals changed: %d %d %d %d",
			NodeAccess, NodeRAG, NodeInference, NodeDataWarehouse)
	}
}

// TestRoleOrdinalsMatchContract 校验 UserRole 与合约 Role 枚举一致。
func TestRoleOrdinalsMatchContract(t *testing.T) {
	if RoleNone != 0 || RoleAuditor != 1 || RoleOperator != 2 || RoleManager != 3 || RoleAdmin != 4 {
		t.Fatalf("UserRole ordinals changed: %d %d %d %d %d",
			RoleNone, RoleAuditor, RoleOperator, RoleManager, RoleAdmin)
	}
}

// TestDataLevelOrdinalsMatchContract 校验 DataLevel 与合约 DataLevel 枚举一致。
func TestDataLevelOrdinalsMatchContract(t *testing.T) {
	if LevelPublic != 0 || LevelInternal != 1 || LevelConfidential != 2 ||
		LevelSecret != 3 || LevelTopSecret != 4 {
		t.Fatalf("DataLevel ordinals changed: %d %d %d %d %d",
			LevelPublic, LevelInternal, LevelConfidential, LevelSecret, LevelTopSecret)
	}
}

// ==================== 数据结构 ====================

func TestRequestRecordShape(t *testing.T) {
	var rec RequestRecord
	rec.RequestID = "REQ-1"
	rec.NodeHashes[0] = "0xaaa"
	rec.Submitters[3] = "0xnode"
	rec.Timestamps[1] = 1700000000
	rec.SubmitCount = 2

	if rec.NodeHashes[0] != "0xaaa" || rec.NodeHashes[1] != "" {
		t.Error("NodeHashes array indexing is wrong")
	}
	if rec.Submitters[3] != "0xnode" {
		t.Error("Submitters array indexing is wrong")
	}
	if rec.Timestamps[1] != 1700000000 {
		t.Error("Timestamps array indexing is wrong")
	}
}

func TestReconciliationResultJSON(t *testing.T) {
	result := ReconciliationResult{
		RequestID:      "REQ-1",
		Consistent:     false,
		ConsensusHash:  "0xabc",
		AnomalousNodes: []NodeType{NodeRAG},
		ReconciledAt:   1700000100,
	}

	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded["request_id"] != "REQ-1" {
		t.Errorf("request_id = %v", decoded["request_id"])
	}
	if decoded["consistent"] != false {
		t.Errorf("consistent = %v", decoded["consistent"])
	}
	nodes, ok := decoded["anomalous_nodes"].([]interface{})
	if !ok || len(nodes) != 1 {
		t.Fatalf("anomalous_nodes should be a 1-element array, got %v", decoded["anomalous_nodes"])
	}
	if nodes[0].(float64) != float64(NodeRAG) {
		t.Errorf("anomalous node value = %v, want %d", nodes[0], NodeRAG)
	}
}

func TestUserPermissionJSON(t *testing.T) {
	perm := UserPermission{
		Address:        "0xuser",
		Role:           RoleManager,
		MaxAccessLevel: LevelSecret,
		Active:         true,
	}

	raw, err := json.Marshal(perm)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	for _, key := range []string{"address", "role", "max_access_level", "active"} {
		if !strings.Contains(string(raw), `"`+key+`"`) {
			t.Errorf("UserPermission JSON missing key %q: %s", key, string(raw))
		}
	}
}
