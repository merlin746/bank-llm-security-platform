 package contract
 
import (
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
 
 func CalculateNodeHash(requestID string, nodeType model.NodeType, payload string) string {
 	input := fmt.Sprintf("%s|%d|%s", requestID, nodeType, payload)
 	hash := sha256Hash([]byte(input))
 	return "0x" + hex.EncodeToString(hash)
 }
 
 func sha256Hash(data []byte) []byte {
 	// TODO: use crypto/sha256.Sum256
 	return make([]byte, 32)
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
