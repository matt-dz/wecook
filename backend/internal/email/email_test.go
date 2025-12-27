package email

import (
	"strings"
	"testing"
)

func TestNewSMTPSender(t *testing.T) {
	config := Config{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "user@example.com",
		Password: "password",
		From:     "sender@example.com",
	}

	sender := NewSMTPSender(config)
	if sender == nil {
		t.Fatal("expected sender to be created, got nil")
	}

	if sender.config.Host != config.Host {
		t.Errorf("expected host %s, got %s", config.Host, sender.config.Host)
	}
	if sender.config.Port != config.Port {
		t.Errorf("expected port %d, got %d", config.Port, sender.config.Port)
	}
}

func TestBuildMessage(t *testing.T) {
	sender := &SMTPSender{
		config: Config{
			From: "sender@example.com",
		},
	}

	tests := []struct {
		name        string
		to          []string
		subject     string
		body        string
		wantFrom    string
		wantTo      string
		wantSubject string
	}{
		{
			name:        "single recipient",
			to:          []string{"recipient@example.com"},
			subject:     "Test Subject",
			body:        "Test Body",
			wantFrom:    "sender@example.com",
			wantTo:      "recipient@example.com",
			wantSubject: "Test Subject",
		},
		{
			name:        "multiple recipients",
			to:          []string{"recipient1@example.com", "recipient2@example.com"},
			subject:     "Test Subject",
			body:        "Test Body",
			wantFrom:    "sender@example.com",
			wantTo:      "recipient1@example.com, recipient2@example.com",
			wantSubject: "Test Subject",
		},
		{
			name:        "html body",
			to:          []string{"recipient@example.com"},
			subject:     "HTML Email",
			body:        "<h1>Hello</h1><p>This is HTML</p>",
			wantFrom:    "sender@example.com",
			wantTo:      "recipient@example.com",
			wantSubject: "HTML Email",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message := sender.buildMessage(tt.to, tt.subject, tt.body)
			messageStr := string(message)

			if !strings.Contains(messageStr, "From: "+tt.wantFrom) {
				t.Errorf("expected From header '%s', got message: %s", tt.wantFrom, messageStr)
			}

			if !strings.Contains(messageStr, "To: "+tt.wantTo) {
				t.Errorf("expected To header '%s', got message: %s", tt.wantTo, messageStr)
			}

			if !strings.Contains(messageStr, "Subject: "+tt.wantSubject) {
				t.Errorf("expected Subject header '%s', got message: %s", tt.wantSubject, messageStr)
			}

			if !strings.Contains(messageStr, tt.body) {
				t.Errorf("expected body '%s' in message, got: %s", tt.body, messageStr)
			}

			if !strings.Contains(messageStr, "MIME-Version: 1.0") {
				t.Error("expected MIME-Version header")
			}

			if !strings.Contains(messageStr, "Content-Type: text/html; charset=UTF-8") {
				t.Error("expected Content-Type header for HTML")
			}
		})
	}
}

func TestSend_NoRecipients(t *testing.T) {
	sender := &SMTPSender{
		config: Config{
			Host:     "smtp.example.com",
			Port:     587,
			Username: "user@example.com",
			Password: "password",
			From:     "sender@example.com",
		},
	}

	err := sender.Send([]string{}, "Test", "Body")
	if err == nil {
		t.Error("expected error when no recipients provided, got nil")
	}

	if !strings.Contains(err.Error(), "no recipients") {
		t.Errorf("expected 'no recipients' error, got: %v", err)
	}
}

func TestTLSInference(t *testing.T) {
	tests := []struct {
		name      string
		port      int
		expectTLS bool
	}{
		{
			name:      "port 587 uses TLS",
			port:      587,
			expectTLS: true,
		},
		{
			name:      "port 465 uses TLS",
			port:      465,
			expectTLS: true,
		},
		{
			name:      "port 25 does not use TLS",
			port:      25,
			expectTLS: false,
		},
		{
			name:      "custom port does not use TLS",
			port:      2525,
			expectTLS: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{
				Host:     "smtp.example.com",
				Port:     tt.port,
				Username: "user@example.com",
				Password: "password",
				From:     "sender@example.com",
			}

			sender := NewSMTPSender(config)
			usesTLS := sender.config.Port == 587 || sender.config.Port == 465

			if usesTLS != tt.expectTLS {
				t.Errorf("expected TLS usage to be %v for port %d, got %v", tt.expectTLS, tt.port, usesTLS)
			}
		})
	}
}
