package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/chats/go-user-api/internal/messaging/rabbitmq"
	"github.com/rs/zerolog/log"
)

// EmailType defines the type of email to be sent
type EmailType string

const (
	EmailTypeWelcome       EmailType = "welcome"
	EmailTypeResetPassword EmailType = "reset_password"
	EmailTypeNotification  EmailType = "notification"
)

// EmailData contains the data for sending an email
type EmailData struct {
	Type      EmailType              `json:"type"`
	To        string                 `json:"to"`
	Subject   string                 `json:"subject"`
	Template  string                 `json:"template"`
	Variables map[string]interface{} `json:"variables"`
	Timestamp time.Time              `json:"timestamp"`
}

// MailService handles email operations using a job queue
type MailService struct {
	jobQueue *rabbitmq.JobQueue
}

// NewMailService creates a new mail service
func NewMailService(jobQueue *rabbitmq.JobQueue) *MailService {
	return &MailService{
		jobQueue: jobQueue,
	}
}

// SendWelcomeEmail queues a welcome email for a new user
func (s *MailService) SendWelcomeEmail(ctx context.Context, to, username, firstName string) error {
	emailData := EmailData{
		Type:     EmailTypeWelcome,
		To:       to,
		Subject:  "Welcome to go-user-api",
		Template: "welcome.html",
		Variables: map[string]interface{}{
			"username":  username,
			"firstName": firstName,
			"appName":   "go-user-api",
		},
		Timestamp: time.Now(),
	}

	return s.queueEmail(ctx, emailData)
}

// SendPasswordResetEmail queues a password reset email
func (s *MailService) SendPasswordResetEmail(ctx context.Context, to, username, temporaryPassword string) error {
	emailData := EmailData{
		Type:     EmailTypeResetPassword,
		To:       to,
		Subject:  "Password Reset for go-user-api",
		Template: "password_reset.html",
		Variables: map[string]interface{}{
			"username":         username,
			"tempPassword":     temporaryPassword,
			"passwordValidFor": "24 hours",
		},
		Timestamp: time.Now(),
	}

	return s.queueEmail(ctx, emailData)
}

// SendNotification queues a notification email
func (s *MailService) SendNotification(ctx context.Context, to, subject, message string) error {
	emailData := EmailData{
		Type:     EmailTypeNotification,
		To:       to,
		Subject:  subject,
		Template: "notification.html",
		Variables: map[string]interface{}{
			"message": message,
		},
		Timestamp: time.Now(),
	}

	return s.queueEmail(ctx, emailData)
}

// queueEmail queues an email for sending
func (s *MailService) queueEmail(ctx context.Context, emailData EmailData) error {
	jsonData, err := json.Marshal(emailData)
	if err != nil {
		return fmt.Errorf("failed to marshal email data: %w", err)
	}

	err = s.jobQueue.Publish(
		ctx,
		"emails", // Queue name
		jsonData, // Message body
		map[string]interface{}{
			"email_type": string(emailData.Type),
			"to":         emailData.To,
			"timestamp":  emailData.Timestamp.Unix(),
		},
	)

	if err != nil {
		log.Error().Err(err).
			Str("email_type", string(emailData.Type)).
			Str("to", emailData.To).
			Msg("Failed to queue email")
		return fmt.Errorf("failed to queue email: %w", err)
	}

	log.Info().
		Str("email_type", string(emailData.Type)).
		Str("to", emailData.To).
		Msg("Email queued successfully")

	return nil
}

// ProcessEmails starts a worker to process emails from the queue
func (s *MailService) ProcessEmails(ctx context.Context) error {
	return s.jobQueue.Consume(ctx, "emails", func(data []byte, headers map[string]interface{}) error {
		var emailData EmailData
		if err := json.Unmarshal(data, &emailData); err != nil {
			return fmt.Errorf("failed to unmarshal email data: %w", err)
		}

		log.Info().
			Str("email_type", string(emailData.Type)).
			Str("to", emailData.To).
			Str("subject", emailData.Subject).
			Msg("Processing email...")

		// In a real application, this would send an actual email
		// This is just a placeholder for demonstration purposes
		log.Info().
			Str("email_type", string(emailData.Type)).
			Str("to", emailData.To).
			Msg("Email sent successfully")

		return nil
	})
}
