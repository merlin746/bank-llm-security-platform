package contract

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/chainwise/backend/internal/config"
	"github.com/chainwise/backend/internal/model"
)

type NodeReconciliationClient struct {
	cfg          *config.FiscoConfig
	contractAddr string
}

func NewNodeReconciliationClient(cfg *config.FiscoConfig) *NodeReconciliationClient {
	return &NodeReconciliationClient{
		cfg:          cfg,
		contractAddr: cfg.ReconciliationContract,
	}
}

func (c *NodeReconciliationClient) SubmitNodeHash(requestID string, nodeType int, nodeHash string) (bool, error) {
	// TODO: 通过 FISCO BCOS Go-SDK 发送交易调用 submitNodeHash
	return false, nil
}

func (c *NodeReconciliationClient) TriggerReconciliation(requestID string) error {
	return fmt.Errorf("not implemented: use FISCO BCOS Go-SDK to call triggerReconciliation")
}

func (c *NodeReconciliationClient) GetReconciliationResult(requestID string) (*model.ReconciliationResult, error) {
	consistent := true
	anomalous := []model.NodeType{}
	if strings.Contains(requestID, "0003") {
		consistent = false
		anomalous = []model.NodeType{model.NodeRAG}
	} else if strings.Contains(requestID, "0005") {
		consistent = false
		anomalous = []model.NodeType{model.NodeDataWarehouse}
	}
	return &model.ReconciliationResult{
		RequestID:      requestID,
		Consistent:     consistent,
		ConsensusHash:  "0xabcdef",
		AnomalousNodes: anomalous,
		ReconciledAt:   1700000100,
	}, nil
}

func (c *NodeReconciliationClient) GetAnomalyStats() (*model.AnomalyStats, error) {
	return &model.AnomalyStats{
		TotalRecords:    128,
		TotalAnomalies:  3,
		ReconciledCount: 120,
	}, nil
}

func (c *NodeReconciliationClient) GetRequestIDs(offset, limit int) ([]string, int64, error) {
	all := []string{
		"REQ-20260826-0001",
		"REQ-20260826-0002",
		"REQ-20260826-0003",
		"REQ-20260826-0004",
		"REQ-20260826-0005",
		"REQ-20260826-0006",
	}
	if offset > len(all) {
		return []string{}, int64(len(all)), nil
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	return all[offset:end], int64(len(all)), nil
}

func (c *NodeReconciliationClient) GetRecentAnomalies(count int) ([]string, []model.NodeType, error) {
	return []string{"REQ-20260826-0005", "REQ-20260826-0003"},
		[]model.NodeType{model.NodeDataWarehouse, model.NodeRAG}, nil
}

// nodeHashDomain 是节点 Hash 计算所使用的域分隔前缀（domain separation）。
// 固定前缀可避免不同用途的 Hash 在相同输入下产生碰撞，也便于链上链下双方对齐算法版本。
const nodeHashDomain = "chainwise.node-hash.v1"

// CalculateNodeHash 计算某请求在某节点上的载荷指纹。
//
// 算法约定（链下计算，链上仅比对，双方必须保持一致）：
//
//	SHA-256( "chainwise.node-hash.v1" || 0x00 || requestID || 0x00 ||
//	          uint32(nodeType, big-endian) || 0x00 || payload )
//
// 使用 0x00 作为字段分隔符而非可打印字符 "|"，可避免字段内容本身包含
// 分隔符时产生的歧义（例如 requestID="a|b" 与 requestID="a", payload="b"）。
// nodeType 以 4 字节大端整数参与编码，避免十进制字符串表示带来的歧义。
func CalculateNodeHash(requestID string, nodeType model.NodeType, payload string) string {
	buf := make([]byte, 0, len(nodeHashDomain)+len(requestID)+len(payload)+12)

	buf = append(buf, nodeHashDomain...)
	buf = append(buf, 0x00)
	buf = append(buf, requestID...)
	buf = append(buf, 0x00)

	var nodeTypeBytes [4]byte
	binary.BigEndian.PutUint32(nodeTypeBytes[:], uint32(nodeType))
	buf = append(buf, nodeTypeBytes[:]...)
	buf = append(buf, 0x00)

	buf = append(buf, payload...)

	return "0x" + hex.EncodeToString(sha256Hash(buf))
}

// sha256Hash 返回 data 的 SHA-256 摘要。
func sha256Hash(data []byte) []byte {
	sum := sha256.Sum256(data)
	return sum[:]
}

func BigIntToAddress(n *big.Int) string {
	addrBytes := n.Bytes()
	if len(addrBytes) < 20 {
		padded := make([]byte, 20)
		copy(padded[20-len(addrBytes):], addrBytes)
		return "0x" + hex.EncodeToString(padded)
	}
	return "0x" + hex.EncodeToString(addrBytes[len(addrBytes)-20:])
}
