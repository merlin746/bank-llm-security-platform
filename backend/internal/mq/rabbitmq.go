 package mq
 
 import (
 	"context"
 	"encoding/json"
 	"fmt"
 	"log"
 	"time"
 
 	amqp "github.com/rabbitmq/amqp091-go"
 
 	"github.com/chainwise/backend/internal/config"
 )
 
 type MessageHandler func(ctx context.Context, body []byte) error
 
 type MQClient struct {
 	conn    *amqp.Connection
 	channel *amqp.Channel
 	cfg     *config.RabbitMQConfig
 }
 
 type HashSubmitMessage struct {
 	RequestID string `json:"request_id"`
 	NodeType  int    `json:"node_type"`
 	NodeHash  string `json:"node_hash"`
 	Submitter string `json:"submitter"`
 	Timestamp int64  `json:"timestamp"`
 }
 
 type ReconSummaryMessage struct {
 	RequestID     string `json:"request_id"`
 	Consistent    bool   `json:"consistent"`
 	ConsensusHash string `json:"consensus_hash"`
 	AnomalyNodes  []int  `json:"anomaly_nodes"`
 	ReconciledAt  int64  `json:"reconciled_at"`
 }
 
 func New(cfg *config.RabbitMQConfig) (*MQClient, error) {
 	conn, err := amqp.Dial(cfg.URL)
 	if err != nil {
 		return nil, fmt.Errorf("rabbitmq dial failed: %w", err)
 	}
 
 	ch, err := conn.Channel()
 	if err != nil {
 		conn.Close()
 		return nil, fmt.Errorf("rabbitmq channel failed: %w", err)
 	}
 
 	if err := ch.ExchangeDeclare(
 		cfg.Exchange, "direct", true, false, false, false, nil,
 	); err != nil {
 		ch.Close()
 		conn.Close()
 		return nil, fmt.Errorf("exchange declare failed: %w", err)
 	}
 
 	_, err = ch.QueueDeclare(
 		cfg.Queue, true, false, false, false, nil,
 	)
 	if err != nil {
 		ch.Close()
 		conn.Close()
 		return nil, fmt.Errorf("queue declare failed: %w", err)
 	}
 
 	if err := ch.QueueBind(
 		cfg.Queue, cfg.RoutingKey, cfg.Exchange, false, nil,
 	); err != nil {
 		ch.Close()
 		conn.Close()
 		return nil, fmt.Errorf("queue bind failed: %w", err)
 	}
 
 	return &MQClient{conn: conn, channel: ch, cfg: cfg}, nil
 }
 
 func (m *MQClient) Close() error {
 	if err := m.channel.Close(); err != nil {
 		return err
 	}
 	return m.conn.Close()
 }
 
 func (m *MQClient) PublishHashSubmit(ctx context.Context, msg *HashSubmitMessage) error {
 	body, err := json.Marshal(msg)
 	if err != nil {
 		return err
 	}
 	return m.channel.PublishWithContext(ctx,
 		m.cfg.Exchange, m.cfg.RoutingKey,
 		false, false,
 		amqp.Publishing{
 			ContentType:  "application/json",
 			DeliveryMode: amqp.Persistent,
 			Timestamp:    time.Now(),
 			Body:         body,
 		},
 	)
 }
 
 func (m *MQClient) PublishReconSummary(ctx context.Context, msg *ReconSummaryMessage) error {
 	body, err := json.Marshal(msg)
 	if err != nil {
 		return err
 	}
 	return m.channel.PublishWithContext(ctx,
 		m.cfg.Exchange, "recon.summary",
 		false, false,
 		amqp.Publishing{
 			ContentType:  "application/json",
 			DeliveryMode: amqp.Persistent,
 			Timestamp:    time.Now(),
 			Body:         body,
 		},
 	)
 }
 
 func (m *MQClient) Consume(ctx context.Context, queue string, handler MessageHandler) error {
 	msgs, err := m.channel.Consume(
 		queue, "", false, false, false, false, nil,
 	)
 	if err != nil {
 		return fmt.Errorf("consume failed: %w", err)
 	}
 
 	go func() {
 		for {
 			select {
 			case <-ctx.Done():
 				log.Println("[MQ] consumer stopped: context canceled")
 				return
 			case msg, ok := <-msgs:
 				if !ok {
 					log.Println("[MQ] consumer channel closed")
 					return
 				}
 				if err := handler(ctx, msg.Body); err != nil {
 					log.Printf("[MQ] handler error: %v, rejecting", err)
 					msg.Nack(false, true)
 					continue
 				}
 				msg.Ack(false)
 			}
 		}
 	}()
 
 	return nil
 }
