// Package email provides functionality for sending emails via SMTP.
package email

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strconv"
	"strings"
)

// Sender defines the interface for sending emails.
type Sender interface {
	Send(to []string, subject, body string) error
}

// TLSMode controls how TLS is negotiated with the SMTP server.
type TLSMode string

const (
	TLSModeAuto     TLSMode = "auto"
	TLSModeStartTLS TLSMode = "starttls"
	TLSModeImplicit TLSMode = "implicit"
	TLSModeNone     TLSMode = "none"
)

const (
	port465 = 465
	port587 = 587
)

// ParseTLSMode normalizes the TLS mode and returns an error if it is invalid.
func ParseTLSMode(mode string) (TLSMode, error) {
	if mode == "" {
		return TLSModeAuto, nil
	}

	normalized := TLSMode(strings.ToLower(mode))
	switch normalized {
	case TLSModeAuto, TLSModeStartTLS, TLSModeImplicit, TLSModeNone:
		return normalized, nil
	default:
		return TLSModeAuto, fmt.Errorf("invalid TLS mode %q", mode)
	}
}

// Config holds the SMTP configuration.
// TLS is automatically determined based on the port when TLSMode is set to auto:
// - Port 587: StartTLS is used.
// - Port 465: Implicit TLS is used.
// - Other ports: TLS is disabled.
type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	TLSMode  TLSMode

	// SkipTLSVerification disables certificate verification. This should only be
	// used for development scenarios with self-signed certificates.
	SkipTLSVerification bool
}

// SMTPSender implements the Sender interface using SMTP.
type SMTPSender struct {
	config Config
}

// NewSMTPSender creates a new SMTP email sender.
func NewSMTPSender(config Config) *SMTPSender {
	if config.TLSMode == "" {
		config.TLSMode = TLSModeAuto
	}

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

	switch s.resolveTLSMode() {
	case TLSModeImplicit:
		return s.sendWithImplicitTLS(addr, auth, to, message)
	case TLSModeStartTLS:
		return s.sendWithStartTLS(addr, auth, to, message)
	}

	return s.sendWithoutTLS(addr, auth, to, message)
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

func (s *SMTPSender) sendWithImplicitTLS(addr string, auth smtp.Auth, to []string, message []byte) error {
	conn, err := tls.Dial("tcp", addr, s.tlsConfig())
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

	return s.authenticateAndSend(client, auth, to, message)
}

func (s *SMTPSender) sendWithStartTLS(addr string, auth smtp.Auth, to []string, message []byte) error {
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer func() { _ = client.Close() }()

	if err := client.Hello(s.config.Host); err != nil {
		return fmt.Errorf("failed to introduce client: %w", err)
	}

	if ok, _ := client.Extension("STARTTLS"); !ok {
		return fmt.Errorf("SMTP server does not support STARTTLS")
	}

	if err := client.StartTLS(s.tlsConfig()); err != nil {
		return fmt.Errorf("failed to start TLS: %w", err)
	}

	return s.authenticateAndSend(client, auth, to, message)
}

func (s *SMTPSender) sendWithoutTLS(addr string, auth smtp.Auth, to []string, message []byte) error {
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer func() { _ = client.Close() }()

	return s.authenticateAndSend(client, auth, to, message)
}

func (s *SMTPSender) authenticateAndSend(client *smtp.Client, auth smtp.Auth, to []string, message []byte) error {
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("failed to authenticate: %w", err)
	}

	if err := client.Mail(s.config.From); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to set recipient %s: %w", recipient, err)
		}
	}

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

func (s *SMTPSender) resolveTLSMode() TLSMode {
	switch strings.ToLower(string(s.config.TLSMode)) {
	case string(TLSModeStartTLS):
		return TLSModeStartTLS
	case string(TLSModeImplicit):
		return TLSModeImplicit
	case string(TLSModeNone):
		return TLSModeNone
	default:
		if s.config.Port == port465 {
			return TLSModeImplicit
		}
		if s.config.Port == port587 {
			return TLSModeStartTLS
		}

		return TLSModeNone
	}
}

func (s *SMTPSender) tlsConfig() *tls.Config {
	return &tls.Config{
		ServerName:         s.config.Host,
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: s.config.SkipTLSVerification,
	}
}
