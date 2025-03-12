package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/chats/go-user-api/config"
	"github.com/rs/zerolog/log"
	"github.com/streadway/amqp"
)

// JobQueue represents a RabbitMQ-based job queue
type JobQueue struct {
	conn         *amqp.Connection
	channel      *amqp.Channel
	exchangeName string
}

// JobHandler is a function that processes a job
type JobHandler func(data []byte, headers map[string]interface{}) error

// NewJobQueue creates a new JobQueue
func NewJobQueue(cfg *config.Config) (*JobQueue, error) {
	// Connect to RabbitMQ
	conn, err := amqp.Dial(cfg.RabbitMQURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	// Create a channel
	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open a channel: %w", err)
	}

	// Declare the exchange
	exchangeName := "go-user-api.jobs"
	err = channel.ExchangeDeclare(
		exchangeName, // name
		"direct",     // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	return &JobQueue{
		conn:         conn,
		channel:      channel,
		exchangeName: exchangeName,
	}, nil
}

// ensureQueue ensures a queue exists and is bound to the exchange
func (q *JobQueue) ensureQueue(queueName string) error {
	// Declare the queue
	_, err := q.channel.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue %s: %w", queueName, err)
	}

	// Bind the queue to the exchange
	err = q.channel.QueueBind(
		queueName,      // queue name
		queueName,      // routing key
		q.exchangeName, // exchange
		false,          // no-wait
		nil,            // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to bind queue %s to exchange: %w", queueName, err)
	}

	return nil
}

// Publish publishes a message to the job queue
func (q *JobQueue) Publish(ctx context.Context, queueName string, data []byte, headers map[string]interface{}) error {
	// Ensure the queue exists
	if err := q.ensureQueue(queueName); err != nil {
		return err
	}

	// Add timestamp to headers if not present
	if _, ok := headers["timestamp"]; !ok {
		headers["timestamp"] = time.Now().Unix()
	}

	// Create AMQP headers table
	amqpHeaders := amqp.Table{}
	for k, v := range headers {
		amqpHeaders[k] = v
	}

	// Publish the message
	err := q.channel.Publish(
		q.exchangeName, // exchange
		queueName,      // routing key
		false,          // mandatory
		false,          // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Headers:      amqpHeaders,
			Body:         data,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish message to queue %s: %w", queueName, err)
	}

	return nil
}

// PublishJSON publishes a JSON-serializable object to the job queue
func (q *JobQueue) PublishJSON(ctx context.Context, queueName string, obj interface{}, headers map[string]interface{}) error {
	// Serialize the object to JSON
	data, err := json.Marshal(obj)
	if err != nil {
		return fmt.Errorf("failed to marshal object to JSON: %w", err)
	}

	return q.Publish(ctx, queueName, data, headers)
}

// Consume consumes messages from a queue
func (q *JobQueue) Consume(ctx context.Context, queueName string, handler JobHandler) error {
	// Ensure the queue exists
	if err := q.ensureQueue(queueName); err != nil {
		return err
	}

	// Set prefetch count (QoS)
	err := q.channel.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	)
	if err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	// Start consuming messages
	msgs, err := q.channel.Consume(
		queueName, // queue
		"",        // consumer
		false,     // auto-ack
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	log.Info().Str("queue", queueName).Msg("Started consuming messages from queue")

	// Process messages
	for {
		select {
		case <-ctx.Done():
			log.Info().Str("queue", queueName).Msg("Context canceled, stopping consumer")
			return nil
		case msg, ok := <-msgs:
			if !ok {
				log.Warn().Str("queue", queueName).Msg("Channel closed, stopping consumer")
				return nil
			}

			// Convert headers to map
			headers := make(map[string]interface{})
			for k, v := range msg.Headers {
				headers[k] = v
			}

			// Process the message
			err := handler(msg.Body, headers)
			if err != nil {
				log.Error().Err(err).
					Str("queue", queueName).
					Interface("headers", headers).
					Msg("Failed to process message")

				// Nack the message and requeue
				if err := msg.Nack(false, true); err != nil {
					log.Error().Err(err).
						Str("queue", queueName).
						Msg("Failed to nack message")
				}
			} else {
				// Acknowledge the message
				if err := msg.Ack(false); err != nil {
					log.Error().Err(err).
						Str("queue", queueName).
						Msg("Failed to ack message")
				}
			}
		}
	}
}

// Close closes the connection to RabbitMQ
func (q *JobQueue) Close() error {
	// Close the channel and connection
	if q.channel != nil {
		if err := q.channel.Close(); err != nil {
			log.Error().Err(err).Msg("Failed to close RabbitMQ channel")
		}
	}

	if q.conn != nil {
		if err := q.conn.Close(); err != nil {
			return fmt.Errorf("failed to close RabbitMQ connection: %w", err)
		}
	}

	return nil
}
