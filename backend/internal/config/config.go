 package config
 
 import (
 	"os"
 
 	"gopkg.in/yaml.v3"
 )
 
 type Config struct {
 	Server    ServerConfig    `yaml:"server"`
 	Redis     RedisConfig     `yaml:"redis"`
 	RabbitMQ  RabbitMQConfig  `yaml:"rabbitmq"`
 	Fisco     FiscoConfig     `yaml:"fisco"`
 	Log       LogConfig       `yaml:"log"`
 }
 
 type ServerConfig struct {
 	Port int    `yaml:"port"`
 	Mode string `yaml:"mode"`
 }
 
 type RedisConfig struct {
 	Addr               string `yaml:"addr"`
 	Password           string `yaml:"password"`
 	DB                 int    `yaml:"db"`
 	PolicyCacheTTLSecs int    `yaml:"policy_cache_ttl_seconds"`
 }
 
 type RabbitMQConfig struct {
 	URL        string `yaml:"url"`
 	Exchange   string `yaml:"exchange"`
 	Queue      string `yaml:"queue"`
 	RoutingKey string `yaml:"routing_key"`
 }
 
 type FiscoConfig struct {
 	ChainID                  int    `yaml:"chain_id"`
 	GroupID                  int    `yaml:"group_id"`
 	NodeEndpoint             string `yaml:"node_endpoint"`
 	AccessControlContract    string `yaml:"access_control_contract"`
 	CompliancePolicyContract string `yaml:"compliance_policy_contract"`
 	ReconciliationContract   string `yaml:"reconciliation_contract"`
 }
 
 type LogConfig struct {
 	Level string `yaml:"level"`
 	File  string `yaml:"file"`
 }
 
 func Load(path string) (*Config, error) {
 	data, err := os.ReadFile(path)
 	if err != nil {
 		return nil, err
 	}
 
 	var cfg Config
 	if err := yaml.Unmarshal(data, &cfg); err != nil {
 		return nil, err
 	}
 
 	if cfg.Server.Port == 0 {
 		cfg.Server.Port = 8080
 	}
 	if cfg.Server.Mode == "" {
 		cfg.Server.Mode = "debug"
 	}
 	if cfg.Redis.PolicyCacheTTLSecs == 0 {
 		cfg.Redis.PolicyCacheTTLSecs = 60
 	}
 
 	return &cfg, nil
 }
