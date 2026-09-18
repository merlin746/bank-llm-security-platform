package config

import (
	"os"
	"path/filepath"
	"testing"
)

// writeConfig 在临时目录写入配置文件并返回路径。
func writeConfig(t *testing.T, content string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write temp config failed: %v", err)
	}
	return path
}

func TestLoadFullConfig(t *testing.T) {
	path := writeConfig(t, `
server:
  port: 9090
  mode: release
redis:
  addr: "localhost:6380"
  password: "secret"
  db: 2
  policy_cache_ttl_seconds: 120
rabbitmq:
  url: "amqp://guest:guest@localhost:5672/"
  exchange: "chainwise.reconciliation"
  queue: "node.hash.submit"
  routing_key: "hash.submit"
fisco:
  chain_id: 2
  group_id: 3
  node_endpoint: "http://localhost:8546"
  access_control_contract: "0xaaa"
  compliance_policy_contract: "0xbbb"
  reconciliation_contract: "0xccc"
ai:
  base_url: "http://127.0.0.1:9000"
log:
  level: "debug"
  file: "./logs/x.log"
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Server.Port != 9090 {
		t.Errorf("Server.Port = %d, want 9090", cfg.Server.Port)
	}
	if cfg.Server.Mode != "release" {
		t.Errorf("Server.Mode = %q, want release", cfg.Server.Mode)
	}
	if cfg.Redis.Addr != "localhost:6380" {
		t.Errorf("Redis.Addr = %q", cfg.Redis.Addr)
	}
	if cfg.Redis.Password != "secret" {
		t.Errorf("Redis.Password = %q", cfg.Redis.Password)
	}
	if cfg.Redis.DB != 2 {
		t.Errorf("Redis.DB = %d, want 2", cfg.Redis.DB)
	}
	if cfg.Redis.PolicyCacheTTLSecs != 120 {
		t.Errorf("Redis.PolicyCacheTTLSecs = %d, want 120", cfg.Redis.PolicyCacheTTLSecs)
	}
	if cfg.RabbitMQ.Queue != "node.hash.submit" {
		t.Errorf("RabbitMQ.Queue = %q", cfg.RabbitMQ.Queue)
	}
	if cfg.Fisco.ChainID != 2 || cfg.Fisco.GroupID != 3 {
		t.Errorf("Fisco ids = %d/%d, want 2/3", cfg.Fisco.ChainID, cfg.Fisco.GroupID)
	}
	if cfg.Fisco.AccessControlContract != "0xaaa" {
		t.Errorf("Fisco.AccessControlContract = %q", cfg.Fisco.AccessControlContract)
	}
	if cfg.Fisco.CompliancePolicyContract != "0xbbb" {
		t.Errorf("Fisco.CompliancePolicyContract = %q", cfg.Fisco.CompliancePolicyContract)
	}
	if cfg.Fisco.ReconciliationContract != "0xccc" {
		t.Errorf("Fisco.ReconciliationContract = %q", cfg.Fisco.ReconciliationContract)
	}
	if cfg.AI.BaseURL != "http://127.0.0.1:9000" {
		t.Errorf("AI.BaseURL = %q", cfg.AI.BaseURL)
	}
	if cfg.Log.Level != "debug" || cfg.Log.File != "./logs/x.log" {
		t.Errorf("Log = %+v", cfg.Log)
	}
}

// TestLoadAppliesDefaults 校验缺省值回填逻辑。
func TestLoadAppliesDefaults(t *testing.T) {
	path := writeConfig(t, "server:\n  port: 0\n  mode: \"\"\n")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Server.Port != 8080 {
		t.Errorf("default Server.Port = %d, want 8080", cfg.Server.Port)
	}
	if cfg.Server.Mode != "debug" {
		t.Errorf("default Server.Mode = %q, want debug", cfg.Server.Mode)
	}
	if cfg.Redis.PolicyCacheTTLSecs != 60 {
		t.Errorf("default PolicyCacheTTLSecs = %d, want 60", cfg.Redis.PolicyCacheTTLSecs)
	}
	if cfg.AI.BaseURL != "http://127.0.0.1:8000" {
		t.Errorf("default AI.BaseURL = %q", cfg.AI.BaseURL)
	}
}

// TestLoadEmptyDocumentUsesAllDefaults 空配置文档也应返回可用配置。
func TestLoadEmptyDocumentUsesAllDefaults(t *testing.T) {
	path := writeConfig(t, "")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed for empty config: %v", err)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("Server.Port = %d, want 8080", cfg.Server.Port)
	}
	if cfg.AI.BaseURL == "" {
		t.Error("AI.BaseURL should have a default")
	}
}

func TestLoadMissingFileReturnsError(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "does-not-exist.yaml"))
	if err == nil {
		t.Fatal("expected an error for a missing config file")
	}
}

// TestLoadInvalidYAMLErrors 非法 YAML 应返回错误而不是 panic。
func TestLoadInvalidYAMLErrors(t *testing.T) {
	path := writeConfig(t, "server: [this is: not valid yaml")

	if _, err := Load(path); err == nil {
		t.Fatal("expected an error for invalid YAML")
	}
}
