 package main
 
 import (
 	"context"
 	"encoding/json"
 	"fmt"
 	"log"
 	"net/http"
 	"os"
 	"os/signal"
 	"syscall"
 	"time"
 
 	"github.com/gin-gonic/gin"
 
 	"github.com/chainwise/backend/internal/api"
 	"github.com/chainwise/backend/internal/cache"
 	"github.com/chainwise/backend/internal/config"
 	"github.com/chainwise/backend/internal/contract"
 	"github.com/chainwise/backend/internal/mq"
 )
 
 func main() {
 	// 加载配置
 	cfgPath := "config.yaml"
 	if len(os.Args) > 1 {
 		cfgPath = os.Args[1]
 	}
 
 	cfg, err := config.Load(cfgPath)
 	if err != nil {
 		log.Fatalf("failed to load config: %v", err)
 	}
 
 	// 设置 Gin 模式
 	gin.SetMode(cfg.Server.Mode)
 
 	// 初始化 Redis 缓存
 	cacheClient, err := cache.New(&cfg.Redis)
 	if err != nil {
 		log.Printf("[WARN] redis init failed: %v, running without cache", err)
 		cacheClient = nil
 	} else {
 		defer cacheClient.Close()
 		log.Println("[INFO] redis connected")
 	}
 
 	// 初始化 RabbitMQ
 	mqClient, err := mq.New(&cfg.RabbitMQ)
 	if err != nil {
 		log.Printf("[WARN] rabbitmq init failed: %v, running without MQ", err)
 		mqClient = nil
 	} else {
 		defer mqClient.Close()
 		log.Println("[INFO] rabbitmq connected")
 	}
 
 	// 初始化合约客户端
 	accessCtrlClient := contract.NewAccessControlClient(&cfg.Fisco)
 	complianceClient := contract.NewCompliancePolicyClient(&cfg.Fisco)
 	reconClient := contract.NewNodeReconciliationClient(&cfg.Fisco)
 
 	// 预加载策略到 Redis 缓存
 	if cacheClient != nil && complianceClient != nil {
 		ctx := context.Background()
 		if err := cacheClient.PreloadPolicies(ctx, complianceClient.GetPolicyForCache); err != nil {
 			log.Printf("[WARN] policy preload failed: %v", err)
 		} else {
 			log.Println("[INFO] policies preloaded to cache")
 		}
 	}
 
 	// 启动 MQ 消费服务（异步对账）
 	if mqClient != nil {
 		ctx := context.Background()
 		handler := createHashSubmitHandler(reconClient, cacheClient)
 		if err := mqClient.Consume(ctx, cfg.RabbitMQ.Queue, handler); err != nil {
 			log.Printf("[WARN] mq consumer start failed: %v", err)
 		} else {
 			log.Println("[INFO] mq consumer started")
 		}
 	}
 
 	// 注册路由
 	r := gin.Default()
 	handler := api.NewHandler(cacheClient, accessCtrlClient, complianceClient, reconClient, mqClient)
 	handler.RegisterRoutes(r)
 
 	// 启动 HTTP 服务
 	srv := &http.Server{
 		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
 		Handler: r,
 	}
 
 	// 优雅关闭
 	go func() {
 		quit := make(chan os.Signal, 1)
 		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
 		<-quit
 		log.Println("[INFO] shutting down server...")
 
 		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
 		defer cancel()
 
 		if err := srv.Shutdown(ctx); err != nil {
 			log.Fatalf("server forced to shutdown: %v", err)
 		}
 		log.Println("[INFO] server stopped")
 	}()
 
 	log.Printf("[INFO] chainwise backend starting on :%d", cfg.Server.Port)
 	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
 		log.Fatalf("server error: %v", err)
 	}
 }
 
 // createHashSubmitHandler 创建 MQ 消息处理器：消费 Hash 提交消息并调用合约上链
 func createHashSubmitHandler(reconClient *contract.NodeReconciliationClient, cacheClient *cache.Cache) mq.MessageHandler {
 	return func(ctx context.Context, body []byte) error {
 		var msg mq.HashSubmitMessage
 		if err := json.Unmarshal(body, &msg); err != nil {
 			return fmt.Errorf("unmarshal hash submit message failed: %w", err)
 		}
 
 		log.Printf("[MQ] processing hash submit: request=%s nodeType=%d", msg.RequestID, msg.NodeType)
 
 		// 调用合约提交节点 Hash
 		ready, err := reconClient.SubmitNodeHash(msg.RequestID, msg.NodeType, msg.NodeHash)
 		if err != nil {
 			return fmt.Errorf("submit node hash to chain failed: %w", err)
 		}
 
 		// 如果四节点已全部提交，自动触发对账结果推送
 		if ready {
 			result, err := reconClient.GetReconciliationResult(msg.RequestID)
 			if err == nil && cacheClient != nil {
 				// 推送异常到 Redis 缓存（供大屏展示）
 				if !result.Consistent {
 					cacheClient.PushAnomaly(ctx, result)
 				}
 			}
 			log.Printf("[MQ] reconciliation completed for request=%s consistent=%v", msg.RequestID, result.Consistent)
 		}
 
 		return nil
 	}
 }
