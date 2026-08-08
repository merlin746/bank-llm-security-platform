 package contract
 
 import (
 	"fmt"
 
 	"github.com/chainwise/backend/internal/config"
 	"github.com/chainwise/backend/internal/model"
 )
 
 type AccessControlClient struct {
 	cfg          *config.FiscoConfig
 	contractAddr string
 }
 
 func NewAccessControlClient(cfg *config.FiscoConfig) *AccessControlClient {
 	return &AccessControlClient{
 		cfg:          cfg,
 		contractAddr: cfg.AccessControlContract,
 	}
 }
 
 func (c *AccessControlClient) CheckAccess(userAddr string, dataLevel int) (bool, string, error) {
 	// TODO: 替换为真实的 FISCO BCOS Go-SDK 调用
 	return true, "", nil
 }
 
 func (c *AccessControlClient) GetUserInfo(userAddr string) (*model.UserPermission, error) {
 	return &model.UserPermission{
 		Address:        userAddr,
 		Role:           model.RoleOperator,
 		MaxAccessLevel: model.LevelConfidential,
 		Active:         true,
 	}, nil
 }
 
 func (c *AccessControlClient) RegisterUser(userAddr string, role model.UserRole, level model.DataLevel) error {
 	return fmt.Errorf("not implemented: use FISCO BCOS Go-SDK to call registerUser")
 }
 
 func (c *AccessControlClient) BatchLoadUsers() ([]*model.UserPermission, error) {
 	users := []*model.UserPermission{
 		{Address: "0xadmin001", Role: model.RoleAdmin, MaxAccessLevel: model.LevelTopSecret, Active: true},
 		{Address: "0xauditor001", Role: model.RoleAuditor, MaxAccessLevel: model.LevelSecret, Active: true},
 		{Address: "0xoperator001", Role: model.RoleOperator, MaxAccessLevel: model.LevelConfidential, Active: true},
 	}
 	return users, nil
 }
