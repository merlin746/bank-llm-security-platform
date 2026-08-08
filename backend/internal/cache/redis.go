 package cache
 
 import (
 	"context"
 	"encoding/json"
 	"fmt"
 	"time"
 
 	"github.com/go-redis/redis/v8"
 
 	"github.com/chainwise/backend/internal/config"
 )
 
 // Cache 封装 Redis 操作，提供策略缓存预加载与毫秒级查询能力
 type Cache struct {
 	client *redis.Client
 	ttl    time.Duration
 }
 
 // 缓存 Key 前缀
 const (
 	KeyUserPermPrefix   = "chainwise:user:perm:"     // 用户权限缓存
 	KeyPolicyPrefix     = "chainwise:policy:active"   // 当前生效策略缓存
 	KeyRateLimitPrefix  = "chainwise:ratelimit:"      // 限流计数
 	KeyAnomalyPrefix    = "chainwise:anomaly:recent"  // 近期异常列表
 )
 
 func New(cfg *config.RedisConfig) (*Cache, error) {
 	client := redis.NewClient(&redis.Options{
 		Addr:     cfg.Addr,
 		Password: cfg.Password,
 		DB:       cfg.DB,
 	})
 
 	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
 	defer cancel()
 
 	if err := client.Ping(ctx).Err(); err != nil {
 		return nil, fmt.Errorf("redis connect failed: %w", err)
 	}
 
 	return &Cache{
 		client: client,
 		ttl:    time.Duration(cfg.PolicyCacheTTLSecs) * time.Second,
 	}, nil
 }
 
 func (c *Cache) Close() error {
 	return c.client.Close()
 }
 
 // ==================== 用户权限缓存 ====================
 
 // SetUserPermission 缓存用户权限信息
 func (c *Cache) SetUserPermission(ctx context.Context, address string, perm interface{}) error {
 	data, err := json.Marshal(perm)
 	if err != nil {
 		return err
 	}
 	return c.client.Set(ctx, KeyUserPermPrefix+address, data, c.ttl).Err()
 }
 
 // GetUserPermission 获取用户权限缓存
 func (c *Cache) GetUserPermission(ctx context.Context, address string, dest interface{}) error {
 	data, err := c.client.Get(ctx, KeyUserPermPrefix+address).Bytes()
 	if err != nil {
 		return err
 	}
 	return json.Unmarshal(data, dest)
 }
 
 // ==================== 策略缓存 ====================
 
 // SetActivePolicy 缓存当前生效策略版本及其规则
 func (c *Cache) SetActivePolicy(ctx context.Context, policy interface{}) error {
 	data, err := json.Marshal(policy)
 	if err != nil {
 		return err
 	}
 	return c.client.Set(ctx, KeyPolicyPrefix, data, c.ttl).Err()
 }
 
 // GetActivePolicy 获取缓存中的当前策略
 func (c *Cache) GetActivePolicy(ctx context.Context, dest interface{}) error {
 	data, err := c.client.Get(ctx, KeyPolicyPrefix).Bytes()
 	if err != nil {
 		return err
 	}
 	return json.Unmarshal(data, dest)
 }
 
 // ==================== 链上策略预加载 ====================
 
 // PreloadPolicies 从链上拉取最新策略并写入缓存（由启动时或定时任务调用）
 func (c *Cache) PreloadPolicies(ctx context.Context, fetchFunc func() (interface{}, error)) error {
 	data, err := fetchFunc()
 	if err != nil {
 		return fmt.Errorf("preload fetch failed: %w", err)
 	}
 
 	if err := c.SetActivePolicy(ctx, data); err != nil {
 		return fmt.Errorf("preload cache set failed: %w", err)
 	}
 
 	return nil
 }
 
 // ==================== 限流相关 ====================
 
 // IncrementRequestCount 递增请求计数，返回当前窗口内的请求次数
 func (c *Cache) IncrementRequestCount(ctx context.Context, userAddr string, windowSec int) (int64, error) {
 	key := KeyRateLimitPrefix + userAddr
 	count, err := c.client.Incr(ctx, key).Result()
 	if err != nil {
 		return 0, err
 	}
 
 	// 第一次递增时设置过期时间
 	if count == 1 {
 		c.client.Expire(ctx, key, time.Duration(windowSec)*time.Second)
 	}
 
 	return count, nil
 }
 
 // GetRequestCount 查询当前窗口内请求次数（仅读取）
 func (c *Cache) GetRequestCount(ctx context.Context, userAddr string) (int64, error) {
 	key := KeyRateLimitPrefix + userAddr
 	return c.client.Get(ctx, key).Int64()
 }
 
 // ==================== 异常缓存 ====================
 
 // PushAnomaly 将异常事件推入列表（限定数量，用于大屏展示）
 func (c *Cache) PushAnomaly(ctx context.Context, anomaly interface{}) error {
 	data, err := json.Marshal(anomaly)
 	if err != nil {
 		return err
 	}
 
 	pipe := c.client.Pipeline()
 	pipe.LPush(ctx, KeyAnomalyPrefix, data)
 	pipe.LTrim(ctx, KeyAnomalyPrefix, 0, 99) // 保留最近 100 条
 	_, err = pipe.Exec(ctx)
 	return err
 }
 
 // GetRecentAnomalies 获取最近的异常事件列表
 func (c *Cache) GetRecentAnomalies(ctx context.Context, count int) ([]string, error) {
 	return c.client.LRange(ctx, KeyAnomalyPrefix, 0, int64(count-1)).Result()
 }
 
 // ==================== 通用操作 ====================
 
 func (c *Cache) Ping(ctx context.Context) error {
 	return c.client.Ping(ctx).Err()
 }
 
 // GetClient 暴露原始 client 供扩展使用
 func (c *Cache) GetClient() *redis.Client {
 	return c.client
 }
