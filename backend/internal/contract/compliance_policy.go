 package contract
 
 import (
 	"github.com/chainwise/backend/internal/config"
 	"github.com/chainwise/backend/internal/model"
 )
 
 type CompliancePolicyClient struct {
 	cfg          *config.FiscoConfig
 	contractAddr string
 }
 
 func NewCompliancePolicyClient(cfg *config.FiscoConfig) *CompliancePolicyClient {
 	return &CompliancePolicyClient{
 		cfg:          cfg,
 		contractAddr: cfg.CompliancePolicyContract,
 	}
 }
 
 func (c *CompliancePolicyClient) GetActiveVersion() (*model.PolicyVersionInfo, error) {
 	return &model.PolicyVersionInfo{
 		VersionID:          1,
 		Description:        "v1.0 初始合规策略",
 		RulesRootHash:      "0xabc123",
 		EffectiveTimestamp: 1700000000,
 		Enacted:            true,
 	}, nil
 }
 
 func (c *CompliancePolicyClient) GetActiveRuleIDs() ([]string, error) {
 	return []string{"RULE_ADS_001", "RULE_INVEST_001", "RULE_PRIVACY_001"}, nil
 }
 
 func (c *CompliancePolicyClient) GetPolicyForCache() (map[string]interface{}, error) {
 	version, err := c.GetActiveVersion()
 	if err != nil {
 		return nil, err
 	}
 	rules, err := c.GetActiveRuleIDs()
 	if err != nil {
 		return nil, err
 	}
 	return map[string]interface{}{
 		"version": version,
 		"rules":   rules,
 	}, nil
 }
