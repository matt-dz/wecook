// Package email provides functionality for sending emails via SMTP.
package email

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strconv"
)

// Sender defines the interface for sending emails.
type Sender interface {
	Send(to []string, subject, body string) error
}

// Config holds the SMTP configuration.
// TLS is automatically determined based on the port:
// - Port 587 or 465: TLS enabled.
// - Other ports: TLS disabled.
type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

// SMTPSender implements the Sender interface using SMTP.
type SMTPSender struct {
	config Config
}

// NewSMTPSender creates a new SMTP email sender.
func NewSMTPSender(config Config) *SMTPSender {
	return &SMTPSender{
		config: config,
	}
}

// Send sends an email to the specified recipients.
func (s *SMTPSender) Send(to []string, subject, body string) error {
	if len(to) == 0 {
		return fmt.Errorf("no recipients specified")
	}

	// Build the message
	message := s.buildMessage(to, subject, body)

	// Create SMTP authentication
	auth := smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host)

	// Build server address
	addr := s.config.Host + ":" + strconv.Itoa(s.config.Port)

	// Send email with TLS if port is 587 or 465
	if s.config.Port == 587 || s.config.Port == 465 {
		return s.sendWithTLS(addr, auth, to, message)
	}

	return smtp.SendMail(addr, auth, s.config.From, to, message)
}

// buildMessage constructs the email message with headers.
func (s *SMTPSender) buildMessage(to []string, subject, body string) []byte {
	headers := make(map[string]string)
	headers["From"] = s.config.From
	headers["To"] = to[0]
	if len(to) > 1 {
		for i := 1; i < len(to); i++ {
			headers["To"] += ", " + to[i]
		}
	}
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"

	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + body

	return []byte(message)
}

// sendWithTLS sends an email using TLS encryption.
func (s *SMTPSender) sendWithTLS(addr string, auth smtp.Auth, to []string, message []byte) error {
	// Create TLS configuration
	tlsConfig := &tls.Config{
		ServerName: s.config.Host,
		MinVersion: tls.VersionTLS12,
	}

	// Connect to the server
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer func() { _ = conn.Close() }()

	// Create SMTP client
	client, err := smtp.NewClient(conn, s.config.Host)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer func() { _ = client.Close() }()

	// Authenticate
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("failed to authenticate: %w", err)
	}

	// Set the sender
	if err := client.Mail(s.config.From); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Set the recipients
	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to set recipient %s: %w", recipient, err)
		}
	}

	// Send the message
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}

	_, err = writer.Write(message)
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close data writer: %w", err)
	}

	return client.Quit()
}
