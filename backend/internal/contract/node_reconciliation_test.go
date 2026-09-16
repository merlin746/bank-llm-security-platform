package contract

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"math/big"
	"strings"
	"testing"

	"github.com/chainwise/backend/internal/config"
	"github.com/chainwise/backend/internal/model"
)

// ==================== 回归测试：sha256Hash 全零缺陷 ====================

// TestSha256HashIsNotAllZero 是 sha256Hash 的回归测试。
//
// 修复前实现为 `return make([]byte, 32)`，对任何输入都返回 32 字节全零，
// 导致 CalculateNodeHash 恒定返回 0x000...0，四节点 Hash 比对完全失效
// （所有请求都会被判定为「一致」），链上存证与定责失去意义。
func TestSha256HashIsNotAllZero(t *testing.T) {
	sum := sha256Hash([]byte("chainwise"))

	if len(sum) != sha256.Size {
		t.Fatalf("sha256Hash should return %d bytes, got %d", sha256.Size, len(sum))
	}

	allZero := true
	for _, b := range sum {
		if b != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Fatal("sha256Hash returned an all-zero digest (the original bug)")
	}
}

// TestSha256HashMatchesStandardLibrary 断言与标准库 crypto/sha256 完全一致。
func TestSha256HashMatchesStandardLibrary(t *testing.T) {
	cases := []string{"", "a", "chainwise", "请求-20260826-0001"}

	for _, input := range cases {
		want := sha256.Sum256([]byte(input))
		got := sha256Hash([]byte(input))

		if hex.EncodeToString(got) != hex.EncodeToString(want[:]) {
			t.Errorf("hash mismatch for %q:\n got %s\nwant %s",
				input, hex.EncodeToString(got), hex.EncodeToString(want[:]))
		}
	}
}

// ==================== CalculateNodeHash ====================

// expectedNodeHash 按文档约定独立推导期望值：
//
//	SHA-256(domain || 0x00 || requestID || 0x00 || uint32(nodeType, BE) || 0x00 || payload)
func expectedNodeHash(requestID string, nodeType model.NodeType, payload string) string {
	var buf []byte
	buf = append(buf, nodeHashDomain...)
	buf = append(buf, 0x00)
	buf = append(buf, requestID...)
	buf = append(buf, 0x00)

	var nt [4]byte
	binary.BigEndian.PutUint32(nt[:], uint32(nodeType))
	buf = append(buf, nt[:]...)
	buf = append(buf, 0x00)
	buf = append(buf, payload...)

	sum := sha256.Sum256(buf)
	return "0x" + hex.EncodeToString(sum[:])
}

func TestCalculateNodeHashMatchesSpec(t *testing.T) {
	cases := []struct {
		requestID string
		nodeType  model.NodeType
		payload   string
	}{
		{"REQ-20260826-0001", model.NodeAccess, "access-payload"},
		{"REQ-20260826-0001", model.NodeRAG, "rag-payload"},
		{"REQ-20260826-0001", model.NodeInference, "inference-payload"},
		{"REQ-20260826-0001", model.NodeDataWarehouse, "warehouse-payload"},
		{"", model.NodeAccess, ""},
		{"REQ-A", model.NodeRAG, ""},
	}

	for _, c := range cases {
		got := CalculateNodeHash(c.requestID, c.nodeType, c.payload)
		want := expectedNodeHash(c.requestID, c.nodeType, c.payload)
		if got != want {
			t.Errorf("CalculateNodeHash(%q, %d, %q):\n got %s\nwant %s",
				c.requestID, c.nodeType, c.payload, got, want)
		}
	}
}

func TestCalculateNodeHashFormat(t *testing.T) {
	hash := CalculateNodeHash("REQ-1", model.NodeAccess, "payload")

	if !strings.HasPrefix(hash, "0x") {
		t.Errorf("hash should be 0x-prefixed, got %s", hash)
	}
	if len(hash) != 2+sha256.Size*2 {
		t.Errorf("hash should be 66 chars (0x + 64 hex), got %d: %s", len(hash), hash)
	}
	if _, err := hex.DecodeString(strings.TrimPrefix(hash, "0x")); err != nil {
		t.Errorf("hash is not valid hex: %v", err)
	}
	if hash == "0x"+strings.Repeat("0", 64) {
		t.Error("hash must not be all zeros (regression)")
	}
}

// TestCalculateNodeHashIsDeterministic 同一输入必须得到同一结果。
func TestCalculateNodeHashIsDeterministic(t *testing.T) {
	first := CalculateNodeHash("REQ-1", model.NodeRAG, "payload")
	for i := 0; i < 10; i++ {
		if got := CalculateNodeHash("REQ-1", model.NodeRAG, "payload"); got != first {
			t.Fatalf("hash is not deterministic: %s != %s", got, first)
		}
	}
}

// TestCalculateNodeHashDiffersByNodeType 同一请求载荷在不同节点上必须得到不同 Hash。
func TestCalculateNodeHashDiffersByNodeType(t *testing.T) {
	base := CalculateNodeHash("REQ-1", model.NodeAccess, "same-payload")

	for _, nt := range []model.NodeType{
		model.NodeRAG, model.NodeInference, model.NodeDataWarehouse,
	} {
		if got := CalculateNodeHash("REQ-1", nt, "same-payload"); got == base {
			t.Errorf("node type %d produced the same hash as ACCESS; node type is not bound", nt)
		}
	}
}

// TestCalculateNodeHashDiffersByPayload 载荷不同则 Hash 必须不同（对账的基础）。
func TestCalculateNodeHashDiffersByPayload(t *testing.T) {
	a := CalculateNodeHash("REQ-1", model.NodeRAG, "payload-a")
	b := CalculateNodeHash("REQ-1", model.NodeRAG, "payload-b")

	if a == b {
		t.Error("different payloads must produce different hashes")
	}
}

// TestCalculateNodeHashNoFieldAmbiguity 验证使用 0x00 分隔符后不存在字段歧义：
// 旧实现用 "|" 拼接，("a|b", "c") 与 ("a", "b|c") 会产生相同输入串。
func TestCalculateNodeHashNoFieldAmbiguity(t *testing.T) {
	ambiguous := CalculateNodeHash("a|b", model.NodeRAG, "c")
	plain := CalculateNodeHash("a", model.NodeRAG, "b|c")

	if ambiguous == plain {
		t.Error("field separator ambiguity: different field splits produced the same hash")
	}
}

// ==================== BigIntToAddress ====================

func TestBigIntToAddress(t *testing.T) {
	cases := []struct {
		name string
		in   *big.Int
		want string
	}{
		{
			name: "zero pads to 20 bytes",
			in:   big.NewInt(0),
			want: "0x" + strings.Repeat("0", 40),
		},
		{
			name: "small value left-padded",
			in:   big.NewInt(1),
			want: "0x" + strings.Repeat("0", 39) + "1",
		},
		{
			name: "exactly 20 bytes",
			in:   new(big.Int).SetBytes([]byte(strings.Repeat("\xff", 20))),
			want: "0x" + strings.Repeat("ff", 20),
		},
		{
			name: "more than 20 bytes truncates to low 20",
			in:   new(big.Int).SetBytes(append([]byte{0xde, 0xad}, []byte(strings.Repeat("\x01", 20))...)),
			want: "0x" + strings.Repeat("01", 20),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := BigIntToAddress(c.in); got != c.want {
				t.Errorf("got %s, want %s", got, c.want)
			}
		})
	}
}

// TestBigIntToAddressAlways20Bytes 输出的十六进制部分必须恒为 40 个字符。
func TestBigIntToAddressAlways20Bytes(t *testing.T) {
	for i := 0; i < 40; i++ {
		n := new(big.Int).Lsh(big.NewInt(1), uint(i))
		got := BigIntToAddress(n)

		if len(got) != 42 {
			t.Errorf("bit %d: expected 42 chars, got %d (%s)", i, len(got), got)
		}
	}
}

// ==================== 客户端桩行为（记录当前契约，便于后续替换时发现变更） ====================

func TestNodeReconciliationClientStubs(t *testing.T) {
	// 构造函数会读取 cfg.ReconciliationContract，因此必须传入非 nil 配置
	client := NewNodeReconciliationClient(&config.FiscoConfig{
		ReconciliationContract: "0x0000000000000000000000000000000000000003",
	})

	ids, total, err := client.GetRequestIDs(0, 3)
	if err != nil {
		t.Fatalf("GetRequestIDs returned error: %v", err)
	}
	if len(ids) != 3 {
		t.Errorf("expected 3 ids, got %d", len(ids))
	}
	if total != 6 {
		t.Errorf("expected total 6, got %d", total)
	}

	// 越界 offset 应返回空切片而不是 panic
	ids, _, err = client.GetRequestIDs(100, 3)
	if err != nil {
		t.Fatalf("GetRequestIDs(out of range) returned error: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("expected empty slice for out-of-range offset, got %d ids", len(ids))
	}

	stats, err := client.GetAnomalyStats()
	if err != nil {
		t.Fatalf("GetAnomalyStats returned error: %v", err)
	}
	if stats == nil {
		t.Fatal("GetAnomalyStats returned nil stats")
	}
}
