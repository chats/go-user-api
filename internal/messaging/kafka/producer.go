package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/chats/go-user-api/config"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
)

// LogType defines the type of log
type LogType string

const (
	LogTypeInfo     LogType = "info"
	LogTypeError    LogType = "error"
	LogTypeWarning  LogType = "warning"
	LogTypeActivity LogType = "activity"
	LogTypeAudit    LogType = "audit"
)

// LogEvent represents a log event
type LogEvent struct {
	Type        LogType                `json:"type"`
	Timestamp   time.Time              `json:"timestamp"`
	ServiceName string                 `json:"service_name"`
	UserID      string                 `json:"user_id,omitempty"`
	RequestID   string                 `json:"request_id,omitempty"`
	Message     string                 `json:"message"`
	Data        map[string]interface{} `json:"data,omitempty"`
	Error       string                 `json:"error,omitempty"`
}

// Producer represents a Kafka producer
type Producer struct {
	writer      *kafka.Writer
	topic       string
	serviceName string
}

// NewProducer creates a new Kafka producer
func NewProducer(cfg *config.Config) (*Producer, error) {
	writer := &kafka.Writer{
		Addr:     kafka.TCP(cfg.KafkaBrokers...),
		Topic:    cfg.KafkaTopic,
		Balancer: &kafka.LeastBytes{},
	}

	return &Producer{
		writer:      writer,
		topic:       cfg.KafkaTopic,
		serviceName: "go-user-api",
	}, nil
}

// SendLogEvent sends a log event to Kafka
func (p *Producer) SendLogEvent(ctx context.Context, event LogEvent) error {
	// Set service name and timestamp
	event.ServiceName = p.serviceName
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Convert event to JSON
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal log event: %w", err)
	}

	// Create Kafka message
	msg := kafka.Message{
		Key:   []byte(event.Type),
		Value: data,
		Time:  event.Timestamp,
		Headers: []kafka.Header{
			{Key: "service", Value: []byte(p.serviceName)},
			{Key: "type", Value: []byte(event.Type)},
		},
	}

	// Send message
	err = p.writer.WriteMessages(ctx, msg)
	if err != nil {
		return fmt.Errorf("failed to write message to Kafka: %w", err)
	}

	return nil
}

// LogInfo sends an info log
func (p *Producer) LogInfo(ctx context.Context, message string, data map[string]interface{}) error {
	return p.SendLogEvent(ctx, LogEvent{
		Type:      LogTypeInfo,
		Timestamp: time.Now(),
		Message:   message,
		Data:      data,
	})
}

// LogError sends an error log
func (p *Producer) LogError(ctx context.Context, message string, err error, data map[string]interface{}) error {
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}

	return p.SendLogEvent(ctx, LogEvent{
		Type:      LogTypeError,
		Timestamp: time.Now(),
		Message:   message,
		Error:     errStr,
		Data:      data,
	})
}

// LogActivity logs a user activity
func (p *Producer) LogActivity(ctx context.Context, userID, requestID, action string, data map[string]interface{}) error {
	if data == nil {
		data = make(map[string]interface{})
	}
	data["action"] = action

	return p.SendLogEvent(ctx, LogEvent{
		Type:      LogTypeActivity,
		Timestamp: time.Now(),
		UserID:    userID,
		RequestID: requestID,
		Message:   fmt.Sprintf("User %s performed action: %s", userID, action),
		Data:      data,
	})
}

// LogAudit logs an audit event
func (p *Producer) LogAudit(ctx context.Context, userID, requestID, resource, action string, data map[string]interface{}) error {
	if data == nil {
		data = make(map[string]interface{})
	}
	data["resource"] = resource
	data["action"] = action

	return p.SendLogEvent(ctx, LogEvent{
		Type:      LogTypeAudit,
		Timestamp: time.Now(),
		UserID:    userID,
		RequestID: requestID,
		Message:   fmt.Sprintf("User %s performed %s on %s", userID, action, resource),
		Data:      data,
	})
}

// Close closes the Kafka producer
func (p *Producer) Close() error {
	if err := p.writer.Close(); err != nil {
		return fmt.Errorf("failed to close Kafka writer: %w", err)
	}
	log.Info().Msg("Kafka producer closed")
	return nil
}
