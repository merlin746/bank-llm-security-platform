 package model
 
 // ==================== 链上数据结构镜像（与 Solidity 合约对应） ====================
 
 // AccessControl 相关
 
 type UserRole int
 
 const (
 	RoleNone     UserRole = 0
 	RoleAuditor  UserRole = 1
 	RoleOperator UserRole = 2
 	RoleManager  UserRole = 3
 	RoleAdmin    UserRole = 4
 )
 
 type DataLevel int
 
 const (
 	LevelPublic      DataLevel = 0
 	LevelInternal     DataLevel = 1
 	LevelConfidential DataLevel = 2
 	LevelSecret       DataLevel = 3
 	LevelTopSecret    DataLevel = 4
 )
 
 type UserPermission struct {
 	Address       string   `json:"address"`
 	Role          UserRole `json:"role"`
 	MaxAccessLevel DataLevel `json:"max_access_level"`
 	Active        bool     `json:"active"`
 }
 
 // CompliancePolicy 相关
 
 type PolicyRule struct {
 	RuleID      string `json:"rule_id"`
 	Category    string `json:"category"`
 	Description string `json:"description"`
 	Severity    int    `json:"severity"`
 	Active      bool   `json:"active"`
 }
 
 type PolicyVersionInfo struct {
 	VersionID          uint64 `json:"version_id"`
 	Description        string `json:"description"`
 	RulesRootHash      string `json:"rules_root_hash"`
 	EffectiveTimestamp uint64 `json:"effective_timestamp"`
 	Enacted            bool   `json:"enacted"`
 }
 
 type HighRiskOperation struct {
 	OperationID  string `json:"operation_id"`
 	Operator     string `json:"operator"`
 	Description  string `json:"description"`
 	Reason       string `json:"reason"`
 	Status       string `json:"status"` // PENDING | APPROVED | REJECTED
 	Reviewer     string `json:"reviewer"`
 	ReviewComent string `json:"review_comment"`
 	CreatedAt    uint64 `json:"created_at"`
 	ReviewedAt   uint64 `json:"reviewed_at"`
 }
 
 // NodeReconciliation 相关
 
 type NodeType int
 
 const (
 	NodeAccess        NodeType = 0
 	NodeRAG           NodeType = 1
 	NodeInference     NodeType = 2
 	NodeDataWarehouse NodeType = 3
 )
 
 func (n NodeType) String() string {
 	switch n {
 	case NodeAccess:
 		return "ACCESS"
 	case NodeRAG:
 		return "RAG"
 	case NodeInference:
 		return "INFERENCE"
 	case NodeDataWarehouse:
 		return "DATA_WAREHOUSE"
 	default:
 		return "UNKNOWN"
 	}
 }
 
 type RequestRecord struct {
 	RequestID  string   `json:"request_id"`
 	NodeHashes [4]string `json:"node_hashes"`
 	Submitters [4]string `json:"submitters"`
 	Timestamps [4]uint64 `json:"timestamps"`
 	SubmitCount int     `json:"submit_count"`
 	Reconciled  bool    `json:"reconciled"`
 	ReconciledAt uint64 `json:"reconciled_at"`
 }
 
 type ReconciliationResult struct {
 	RequestID     string     `json:"request_id"`
 	Consistent    bool       `json:"consistent"`
 	ConsensusHash string     `json:"consensus_hash"`
 	AnomalousNodes []NodeType `json:"anomalous_nodes"`
 	ReconciledAt  uint64     `json:"reconciled_at"`
 }
 
 type AnomalyStats struct {
 	TotalRecords    uint64 `json:"total_records"`
 	TotalAnomalies  uint64 `json:"total_anomalies"`
 	ReconciledCount uint64 `json:"reconciled_count"`
 }
 
 // ==================== API 请求/响应 ====================
 
 type PaginatedRequest struct {
 	Offset int `json:"offset" form:"offset"`
 	Limit  int `json:"limit" form:"limit"`
 }
 
 type PageResponse struct {
 	Items interface{} `json:"items"`
 	Total int64       `json:"total"`
 	Page  int         `json:"page"`
 	Size  int         `json:"size"`
 }
 
 type APIResponse struct {
 	Code    int         `json:"code"`
 	Message string      `json:"message"`
 	Data    interface{} `json:"data,omitempty"`
 }
 
 func Success(data interface{}) APIResponse {
 	return APIResponse{Code: 0, Message: "success", Data: data}
 }
 
 func Error(code int, message string) APIResponse {
 	return APIResponse{Code: code, Message: message}
 }
