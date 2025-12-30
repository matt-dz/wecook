package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigFromEnv(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*testing.T)
		wantError bool
		validate  func(*testing.T, *Config)
	}{
		{
			name: "all defaults",
			setup: func(t *testing.T) {
				t.Setenv("DATABASE_USER", "testuser")
				t.Setenv("DATABASE_PASSWORD", "testpass")
				t.Setenv("DATABASE", "testdb")
			},
			wantError: false,
			validate: func(t *testing.T, c *Config) {
				if c.Env != EnvDev {
					t.Errorf("expected Env %q, got %q", EnvDev, c.Env)
				}
				if c.HostOrigin != "http://localhost:8080" {
					t.Errorf("expected HostOrigin %q, got %q", "http://localhost:8080", c.HostOrigin)
				}
				// AppSecret.Path will be the temp directory we set in setup
				if c.AppSecret.Path == "" {
					t.Error("expected AppSecret.Path to be set")
				}
				if c.AppSecret.Version != "1" {
					t.Errorf("expected AppSecret.Version %q, got %q", "1", c.AppSecret.Version)
				}
				if c.Database.Port != 5432 {
					t.Errorf("expected Database.Port 5432, got %d", c.Database.Port)
				}
				if c.Database.Host != "localhost" {
					t.Errorf("expected Database.Host %q, got %q", "localhost", c.Database.Host)
				}
				if c.Database.User != "testuser" {
					t.Errorf("expected Database.User %q, got %q", "testuser", c.Database.User)
				}
				if c.Database.Password != "testpass" {
					t.Errorf("expected Database.Password %q, got %q", "testpass", c.Database.Password)
				}
				if c.Database.Database != "testdb" {
					t.Errorf("expected Database.Database %q, got %q", "testdb", c.Database.Database)
				}
				if c.Fileserver.Volume != "/data/files" {
					t.Errorf("expected Fileserver.Volume %q, got %q", "/data/files", c.Fileserver.Volume)
				}
				if c.Fileserver.URLPrefix != "/files" {
					t.Errorf("expected Fileserver.URLPrefix %q, got %q", "/files", c.Fileserver.URLPrefix)
				}
				if c.SMTP.Port != 587 {
					t.Errorf("expected SMTP.Port 587, got %d", c.SMTP.Port)
				}
				if c.SMTP.TLSMode != TLSModeAuto {
					t.Errorf("expected SMTP.TLSMode %q, got %q", TLSModeAuto, c.SMTP.TLSMode)
				}
				if c.SMTP.TLSSkipVerify != false {
					t.Errorf("expected SMTP.TLSSkipVerify false, got true")
				}
				// AppSecret.Value should be set by loadAppSecret
				if c.AppSecret.Value == nil {
					t.Error("expected AppSecret.Value to be set, got nil")
				}
			},
		},
		{
			name: "custom environment values",
			setup: func(t *testing.T) {
				t.Setenv("ENV", "PROD")
				t.Setenv("HOST_ORIGIN", "https://example.com")
				t.Setenv("APP_SECRET", "this-is-a-very-long-secret-key-with-more-than-32-bytes")
				t.Setenv("APP_SECRET_PATH", "/custom/path/secret")
				t.Setenv("APP_SECRET_VERSION", "2")
				t.Setenv("DATABASE_USER", "customuser")
				t.Setenv("DATABASE_PASSWORD", "custompass")
				t.Setenv("DATABASE", "customdb")
				t.Setenv("DATABASE_HOST", "db.example.com")
				t.Setenv("DATABASE_PORT", "5433")
				t.Setenv("FILESERVER_VOLUME", "/custom/files")
				t.Setenv("FILESERVER_URL_PREFIX", "/uploads")
				t.Setenv("SMTP_HOST", "smtp.example.com")
				t.Setenv("SMTP_PORT", "465")
				t.Setenv("SMTP_USERNAME", "user@example.com")
				t.Setenv("SMTP_PASSWORD", "smtppass")
				t.Setenv("SMTP_FROM", "noreply@example.com")
				t.Setenv("SMTP_TLS_MODE", "implicit")
				t.Setenv("SMTP_TLS_SKIP_VERIFY", "true")
				t.Setenv("ADMIN_FIRST_NAME", "John")
				t.Setenv("ADMIN_LAST_NAME", "Doe")
				t.Setenv("ADMIN_EMAIL", "admin@example.com")
				t.Setenv("ADMIN_PASSWORD", "SecureP@ss123!")
			},
			wantError: false,
			validate: func(t *testing.T, c *Config) {
				if c.Env != EnvProd {
					t.Errorf("expected Env %q, got %q", EnvProd, c.Env)
				}
				if c.HostOrigin != "https://example.com" {
					t.Errorf("expected HostOrigin %q, got %q", "https://example.com", c.HostOrigin)
				}
				if c.AppSecret.Path != "/custom/path/secret" {
					t.Errorf("expected AppSecret.Path %q, got %q", "/custom/path/secret", c.AppSecret.Path)
				}
				if c.AppSecret.Version != "2" {
					t.Errorf("expected AppSecret.Version %q, got %q", "2", c.AppSecret.Version)
				}
				if c.AppSecret.Value == nil {
					t.Error("expected AppSecret.Value to be set, got nil")
				} else if string(*c.AppSecret.Value) != "this-is-a-very-long-secret-key-with-more-than-32-bytes" {
					t.Errorf("expected AppSecret.Value to match provided value")
				}
				if c.Database.Port != 5433 {
					t.Errorf("expected Database.Port 5433, got %d", c.Database.Port)
				}
				if c.Database.Host != "db.example.com" {
					t.Errorf("expected Database.Host %q, got %q", "db.example.com", c.Database.Host)
				}
				if c.Fileserver.Volume != "/custom/files" {
					t.Errorf("expected Fileserver.Volume %q, got %q", "/custom/files", c.Fileserver.Volume)
				}
				if c.Fileserver.URLPrefix != "/uploads" {
					t.Errorf("expected Fileserver.URLPrefix %q, got %q", "/uploads", c.Fileserver.URLPrefix)
				}
				if c.SMTP.Port != 465 {
					t.Errorf("expected SMTP.Port 465, got %d", c.SMTP.Port)
				}
				if c.SMTP.Host != "smtp.example.com" {
					t.Errorf("expected SMTP.Host %q, got %q", "smtp.example.com", c.SMTP.Host)
				}
				if c.SMTP.TLSMode != TLSModeImplicit {
					t.Errorf("expected SMTP.TLSMode %q, got %q", TLSModeImplicit, c.SMTP.TLSMode)
				}
				if c.SMTP.TLSSkipVerify != true {
					t.Errorf("expected SMTP.TLSSkipVerify true, got false")
				}
				if c.Admin.FirstName != "John" {
					t.Errorf("expected Admin.FirstName %q, got %q", "John", c.Admin.FirstName)
				}
				if c.Admin.Email != "admin@example.com" {
					t.Errorf("expected Admin.Email %q, got %q", "admin@example.com", c.Admin.Email)
				}
			},
		},
		{
			name: "invalid database port",
			setup: func(t *testing.T) {
				t.Setenv("DATABASE_PORT", "invalid")
				t.Setenv("DATABASE_USER", "testuser")
				t.Setenv("DATABASE_PASSWORD", "testpass")
				t.Setenv("DATABASE", "testdb")
			},
			wantError: true,
		},
		{
			name: "invalid SMTP port",
			setup: func(t *testing.T) {
				t.Setenv("SMTP_PORT", "999999")
				t.Setenv("DATABASE_USER", "testuser")
				t.Setenv("DATABASE_PASSWORD", "testpass")
				t.Setenv("DATABASE", "testdb")
			},
			wantError: true,
		},
		{
			name: "invalid TLS skip verify",
			setup: func(t *testing.T) {
				t.Setenv("SMTP_TLS_SKIP_VERIFY", "invalid")
				t.Setenv("DATABASE_USER", "testuser")
				t.Setenv("DATABASE_PASSWORD", "testpass")
				t.Setenv("DATABASE", "testdb")
			},
			wantError: true,
		},
		{
			name: "app secret auto-generation",
			setup: func(t *testing.T) {
				t.Setenv("DATABASE_USER", "testuser")
				t.Setenv("DATABASE_PASSWORD", "testpass")
				t.Setenv("DATABASE", "testdb")
			},
			wantError: false,
			validate: func(t *testing.T, c *Config) {
				if c.AppSecret.Value == nil {
					t.Error("expected AppSecret.Value to be auto-generated, got nil")
				} else if len([]byte(*c.AppSecret.Value)) < 32 {
					t.Errorf("expected AppSecret.Value to be at least 32 bytes, got %d", len([]byte(*c.AppSecret.Value)))
				}
			},
		},
		{
			name: "admin validation - email and password set but missing first name",
			setup: func(t *testing.T) {
				t.Setenv("DATABASE_USER", "testuser")
				t.Setenv("DATABASE_PASSWORD", "testpass")
				t.Setenv("DATABASE", "testdb")
				t.Setenv("ADMIN_EMAIL", "admin@example.com")
				t.Setenv("ADMIN_PASSWORD", "SecureP@ss123!")
				t.Setenv("ADMIN_LAST_NAME", "Doe")
				// ADMIN_FIRST_NAME is missing
			},
			wantError: true, // Should fail validation
		},
		{
			name: "admin validation - email and password set but missing last name",
			setup: func(t *testing.T) {
				t.Setenv("DATABASE_USER", "testuser")
				t.Setenv("DATABASE_PASSWORD", "testpass")
				t.Setenv("DATABASE", "testdb")
				t.Setenv("ADMIN_EMAIL", "admin@example.com")
				t.Setenv("ADMIN_PASSWORD", "SecureP@ss123!")
				t.Setenv("ADMIN_FIRST_NAME", "John")
				// ADMIN_LAST_NAME is missing
			},
			wantError: true, // Should fail validation
		},
		{
			name: "admin validation - email and password set but missing both names",
			setup: func(t *testing.T) {
				t.Setenv("DATABASE_USER", "testuser")
				t.Setenv("DATABASE_PASSWORD", "testpass")
				t.Setenv("DATABASE", "testdb")
				t.Setenv("ADMIN_EMAIL", "admin@example.com")
				t.Setenv("ADMIN_PASSWORD", "SecureP@ss123!")
				// ADMIN_FIRST_NAME and ADMIN_LAST_NAME are missing
			},
			wantError: true, // Should fail validation
		},
		{
			name: "admin validation - only email set (no password) - names not required",
			setup: func(t *testing.T) {
				t.Setenv("DATABASE_USER", "testuser")
				t.Setenv("DATABASE_PASSWORD", "testpass")
				t.Setenv("DATABASE", "testdb")
				t.Setenv("ADMIN_EMAIL", "admin@example.com")
				// No ADMIN_PASSWORD, ADMIN_FIRST_NAME, ADMIN_LAST_NAME
			},
			wantError: false, // Should pass - names only required when both email and password are set
			validate: func(t *testing.T, c *Config) {
				if c.Admin.Email != "admin@example.com" {
					t.Errorf("expected Admin.Email %q, got %q", "admin@example.com", c.Admin.Email)
				}
				if c.Admin.FirstName != "" {
					t.Errorf("expected Admin.FirstName to be empty, got %q", c.Admin.FirstName)
				}
			},
		},
		{
			name: "admin validation - only password set (no email) - names not required",
			setup: func(t *testing.T) {
				t.Setenv("DATABASE_USER", "testuser")
				t.Setenv("DATABASE_PASSWORD", "testpass")
				t.Setenv("DATABASE", "testdb")
				t.Setenv("ADMIN_PASSWORD", "SecureP@ss123!")
				// No ADMIN_EMAIL, ADMIN_FIRST_NAME, ADMIN_LAST_NAME
			},
			wantError: false, // Should pass - names only required when both email and password are set
			validate: func(t *testing.T, c *Config) {
				if c.Admin.Email != "" {
					t.Errorf("expected Admin.Email to be empty, got %q", c.Admin.Email)
				}
			},
		},
		{
			name: "admin validation - all admin fields set correctly",
			setup: func(t *testing.T) {
				t.Setenv("DATABASE_USER", "testuser")
				t.Setenv("DATABASE_PASSWORD", "testpass")
				t.Setenv("DATABASE", "testdb")
				t.Setenv("ADMIN_EMAIL", "admin@example.com")
				t.Setenv("ADMIN_PASSWORD", "SecureP@ss123!")
				t.Setenv("ADMIN_FIRST_NAME", "Jane")
				t.Setenv("ADMIN_LAST_NAME", "Smith")
			},
			wantError: false,
			validate: func(t *testing.T, c *Config) {
				if c.Admin.Email != "admin@example.com" {
					t.Errorf("expected Admin.Email %q, got %q", "admin@example.com", c.Admin.Email)
				}
				if c.Admin.FirstName != "Jane" {
					t.Errorf("expected Admin.FirstName %q, got %q", "Jane", c.Admin.FirstName)
				}
				if c.Admin.LastName != "Smith" {
					t.Errorf("expected Admin.LastName %q, got %q", "Smith", c.Admin.LastName)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use temp directory for app secret
			tempDir := t.TempDir()
			secretPath := filepath.Join(tempDir, "secret")
			t.Setenv("APP_SECRET_PATH", secretPath)

			tt.setup(t)

			config, err := loadConfigFromEnv()

			if tt.wantError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.validate != nil {
				tt.validate(t, &config)
			}
		})
	}
}

func TestLoadConfigFromFile(t *testing.T) {
	tests := []struct {
		name      string
		yaml      interface{} // Can be string or func(*testing.T) string
		wantError bool
		validate  func(*testing.T, *Config)
	}{
		{
			name: "complete config",
			yaml: `
env: PROD
host_origin: https://example.com
app_secret:
  value: this-is-a-very-long-secret-key-with-more-than-32-bytes
  path: /custom/secret
  version: "2"
database:
  host: db.example.com
  port: 5433
  database: proddb
  user: produser
  password: prodpass
fileserver:
  volume: /data/production/files
  url_prefix: /uploads
smtp:
  host: smtp.example.com
  port: 465
  username: smtp@example.com
  password: smtppass
  from: noreply@example.com
  tls_mode: implicit
  tls_skip_verify: true
admin:
  first_name: Admin
  last_name: User
  email: admin@example.com
  password: SecureP@ss123!
`,
			wantError: false,
			validate: func(t *testing.T, c *Config) {
				if c.Env != EnvProd {
					t.Errorf("expected Env %q, got %q", EnvProd, c.Env)
				}
				if c.HostOrigin != "https://example.com" {
					t.Errorf("expected HostOrigin %q, got %q", "https://example.com", c.HostOrigin)
				}
				if c.AppSecret.Version != "2" {
					t.Errorf("expected AppSecret.Version %q, got %q", "2", c.AppSecret.Version)
				}
				if c.Database.Port != 5433 {
					t.Errorf("expected Database.Port 5433, got %d", c.Database.Port)
				}
				if c.SMTP.TLSMode != TLSModeImplicit {
					t.Errorf("expected SMTP.TLSMode %q, got %q", TLSModeImplicit, c.SMTP.TLSMode)
				}
			},
		},
		{
			name: "minimal config with defaults",
			yaml: func(t *testing.T) string {
				tempDir := t.TempDir()
				return fmt.Sprintf(`
app_secret:
  path: %s
database:
  database: testdb
  user: testuser
  password: testpass
`, filepath.Join(tempDir, "secret"))
			},
			wantError: false,
			validate: func(t *testing.T, c *Config) {
				if c.Env != EnvDev {
					t.Errorf("expected default Env %q, got %q", EnvDev, c.Env)
				}
				if c.HostOrigin != "http://localhost:8080" {
					t.Errorf("expected default HostOrigin %q, got %q", "http://localhost:8080", c.HostOrigin)
				}
				if c.AppSecret.Path == "" {
					t.Error("expected AppSecret.Path to be set")
				}
				if c.AppSecret.Version != "1" {
					t.Errorf("expected default AppSecret.Version %q, got %q", "1", c.AppSecret.Version)
				}
				if c.Database.Host != "localhost" {
					t.Errorf("expected default Database.Host %q, got %q", "localhost", c.Database.Host)
				}
				if c.Database.Port != 5432 {
					t.Errorf("expected default Database.Port 5432, got %d", c.Database.Port)
				}
				if c.Fileserver.Volume != "/data/files" {
					t.Errorf("expected default Fileserver.Volume %q, got %q", "/data/files", c.Fileserver.Volume)
				}
				if c.Fileserver.URLPrefix != "/files" {
					t.Errorf("expected default Fileserver.URLPrefix %q, got %q", "/files", c.Fileserver.URLPrefix)
				}
				if c.SMTP.Port != 587 {
					t.Errorf("expected default SMTP.Port 587, got %d", c.SMTP.Port)
				}
				if c.SMTP.TLSMode != TLSModeAuto {
					t.Errorf("expected default SMTP.TLSMode %q, got %q", TLSModeAuto, c.SMTP.TLSMode)
				}
			},
		},
		{
			name:      "invalid YAML",
			yaml:      `{invalid yaml content`,
			wantError: true,
		},
		{
			name: "invalid host origin",
			yaml: `
host_origin: not-a-valid-url
database:
  database: testdb
  user: testuser
  password: testpass
`,
			wantError: true,
		},
		{
			name: "app secret auto-generation from file",
			yaml: func(t *testing.T) string {
				tempDir := t.TempDir()
				return `
app_secret:
  path: ` + filepath.Join(tempDir, "secret") + `
database:
  database: testdb
  user: testuser
  password: testpass
`
			},
			wantError: false,
			validate: func(t *testing.T, c *Config) {
				if c.AppSecret.Value == nil {
					t.Error("expected AppSecret.Value to be auto-generated, got nil")
				} else if len([]byte(*c.AppSecret.Value)) < 32 {
					t.Errorf("expected AppSecret.Value to be at least 32 bytes, got %d", len([]byte(*c.AppSecret.Value)))
				}
			},
		},
		{
			name: "admin validation - email and password set but missing first name",
			yaml: func(t *testing.T) string {
				tempDir := t.TempDir()
				return `
app_secret:
  path: ` + filepath.Join(tempDir, "secret") + `
database:
  database: testdb
  user: testuser
  password: testpass
admin:
  last_name: Doe
  email: admin@example.com
  password: SecureP@ss123!
`
			},
			wantError: true, // Should fail validation
		},
		{
			name: "admin validation - email and password set but missing last name",
			yaml: func(t *testing.T) string {
				tempDir := t.TempDir()
				return `
app_secret:
  path: ` + filepath.Join(tempDir, "secret") + `
database:
  database: testdb
  user: testuser
  password: testpass
admin:
  first_name: John
  email: admin@example.com
  password: SecureP@ss123!
`
			},
			wantError: true, // Should fail validation
		},
		{
			name: "admin validation - email and password set but missing both names",
			yaml: func(t *testing.T) string {
				tempDir := t.TempDir()
				return `
app_secret:
  path: ` + filepath.Join(tempDir, "secret") + `
database:
  database: testdb
  user: testuser
  password: testpass
admin:
  email: admin@example.com
  password: SecureP@ss123!
`
			},
			wantError: true, // Should fail validation
		},
		{
			name: "admin validation - only email set (no password) - names not required",
			yaml: func(t *testing.T) string {
				tempDir := t.TempDir()
				return `
app_secret:
  path: ` + filepath.Join(tempDir, "secret") + `
database:
  database: testdb
  user: testuser
  password: testpass
admin:
  email: admin@example.com
`
			},
			wantError: false, // Should pass - names only required when both email and password are set
			validate: func(t *testing.T, c *Config) {
				if c.Admin.Email != "admin@example.com" {
					t.Errorf("expected Admin.Email %q, got %q", "admin@example.com", c.Admin.Email)
				}
				if c.Admin.FirstName != "" {
					t.Errorf("expected Admin.FirstName to be empty, got %q", c.Admin.FirstName)
				}
			},
		},
		{
			name: "admin validation - only password set (no email) - names not required",
			yaml: func(t *testing.T) string {
				tempDir := t.TempDir()
				return `
app_secret:
  path: ` + filepath.Join(tempDir, "secret") + `
database:
  database: testdb
  user: testuser
  password: testpass
admin:
  password: SecureP@ss123!
`
			},
			wantError: false, // Should pass - names only required when both email and password are set
			validate: func(t *testing.T, c *Config) {
				if c.Admin.Email != "" {
					t.Errorf("expected Admin.Email to be empty, got %q", c.Admin.Email)
				}
			},
		},
		{
			name: "admin validation - all admin fields set correctly",
			yaml: func(t *testing.T) string {
				tempDir := t.TempDir()
				return `
app_secret:
  path: ` + filepath.Join(tempDir, "secret") + `
database:
  database: testdb
  user: testuser
  password: testpass
admin:
  first_name: Jane
  last_name: Smith
  email: admin@example.com
  password: SecureP@ss123!
`
			},
			wantError: false,
			validate: func(t *testing.T, c *Config) {
				if c.Admin.Email != "admin@example.com" {
					t.Errorf("expected Admin.Email %q, got %q", "admin@example.com", c.Admin.Email)
				}
				if c.Admin.FirstName != "Jane" {
					t.Errorf("expected Admin.FirstName %q, got %q", "Jane", c.Admin.FirstName)
				}
				if c.Admin.LastName != "Smith" {
					t.Errorf("expected Admin.LastName %q, got %q", "Smith", c.Admin.LastName)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file with YAML content
			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, "config.yaml")

			// Get YAML content (support both string and function)
			var yamlContent string
			switch v := tt.yaml.(type) {
			case string:
				yamlContent = v
			case func(*testing.T) string:
				yamlContent = v(t)
			default:
				t.Fatalf("unexpected yaml type: %T", tt.yaml)
			}

			if err := os.WriteFile(configPath, []byte(yamlContent), 0o644); err != nil {
				t.Fatalf("failed to write test config file: %v", err)
			}

			config, err := loadConfigFromFile(configPath)

			if tt.wantError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.validate != nil {
				tt.validate(t, &config)
			}
		})
	}
}

func TestLoadConfigFromFile_FileNotFound(t *testing.T) {
	_, err := loadConfigFromFile("/nonexistent/config.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}

func TestLoadAppSecret(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*testing.T) *Config
		wantError bool
		validate  func(*testing.T, *Config)
	}{
		{
			name: "secret already set - no file operations",
			setup: func(t *testing.T) *Config {
				secretValue := AppSecretValue("existing-secret-that-is-more-than-32-bytes-long")
				return &Config{
					AppSecret: AppSecret{
						Value:   &secretValue,
						Path:    "/should/not/be/accessed",
						Version: "1",
					},
				}
			},
			wantError: false,
			validate: func(t *testing.T, c *Config) {
				if c.AppSecret.Value == nil {
					t.Error("expected AppSecret.Value to remain set, got nil")
				} else if string(*c.AppSecret.Value) != "existing-secret-that-is-more-than-32-bytes-long" {
					t.Error("AppSecret.Value should not have changed")
				}
			},
		},
		{
			name: "generate new secret - file does not exist",
			setup: func(t *testing.T) *Config {
				tempDir := t.TempDir()
				secretPath := filepath.Join(tempDir, "newsecret")
				return &Config{
					AppSecret: AppSecret{
						Value:   nil,
						Path:    secretPath,
						Version: "1",
					},
				}
			},
			wantError: false,
			validate: func(t *testing.T, c *Config) {
				if c.AppSecret.Value == nil {
					t.Error("expected AppSecret.Value to be generated, got nil")
				} else if len([]byte(*c.AppSecret.Value)) < 32 {
					t.Errorf("expected generated secret to be at least 32 bytes, got %d", len([]byte(*c.AppSecret.Value)))
				}

				// Verify file was created
				if _, err := os.Stat(c.AppSecret.Path); os.IsNotExist(err) {
					t.Error("expected secret file to be created, but it doesn't exist")
				}

				// Verify file contents match config
				contents, err := os.ReadFile(c.AppSecret.Path)
				if err != nil {
					t.Fatalf("failed to read secret file: %v", err)
				}
				if string(contents) != string(*c.AppSecret.Value) {
					t.Error("secret file contents don't match config value")
				}
			},
		},
		{
			name: "load existing secret from file",
			setup: func(t *testing.T) *Config {
				tempDir := t.TempDir()
				secretPath := filepath.Join(tempDir, "existingsecret")

				// Create existing secret file
				existingSecret := "existing-file-secret-that-is-more-than-32-bytes"
				if err := os.WriteFile(secretPath, []byte(existingSecret), 0o644); err != nil {
					t.Fatalf("failed to create test secret file: %v", err)
				}

				return &Config{
					AppSecret: AppSecret{
						Value:   nil,
						Path:    secretPath,
						Version: "1",
					},
				}
			},
			wantError: false,
			validate: func(t *testing.T, c *Config) {
				if c.AppSecret.Value == nil {
					t.Error("expected AppSecret.Value to be loaded from file, got nil")
				} else if string(*c.AppSecret.Value) != "existing-file-secret-that-is-more-than-32-bytes" {
					t.Errorf("expected AppSecret.Value to match file contents, got %q", string(*c.AppSecret.Value))
				}
			},
		},
		{
			name: "error - path is directory",
			setup: func(t *testing.T) *Config {
				tempDir := t.TempDir()
				// Use the directory itself, not a file within it
				return &Config{
					AppSecret: AppSecret{
						Value:   nil,
						Path:    tempDir,
						Version: "1",
					},
				}
			},
			wantError: true,
		},
		{
			name: "error - cannot create file in nonexistent directory",
			setup: func(t *testing.T) *Config {
				return &Config{
					AppSecret: AppSecret{
						Value:   nil,
						Path:    "/nonexistent/directory/secret",
						Version: "1",
					},
				}
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := tt.setup(t)

			err := loadAppSecret(config)

			if tt.wantError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.validate != nil {
				tt.validate(t, config)
			}
		})
	}
}
