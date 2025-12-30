// Package config contains utilities for loading configs
package config

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/goccy/go-yaml"

	"github.com/go-playground/validator/v10"
	"github.com/matt-dz/wecook/internal/password"
)

const (
	PreferenceID = 1
)

const (
	configFilePath     = "/data/wecook.yaml"
	appSecretBytes     = 32
	appSecretFilePerms = 0o600
)

const (
	EnvProd = "PROD"
	EnvDev  = "DEV"
)

type TLSMode string

const (
	TLSModeAuto     TLSMode = "auto"
	TLSModeStartTLS TLSMode = "starttls"
	TLSModeImplicit TLSMode = "implicit"
	TLSModeNone     TLSMode = "none"
)

func (t TLSMode) Validate() error {
	switch t {
	case TLSModeAuto, TLSModeStartTLS, TLSModeImplicit, TLSModeNone:
		return nil
	}
	return fmt.Errorf("unknown tls mode: %q", t)
}

type AdminPassword string

func (a AdminPassword) Validate() error {
	return password.ValidatePassword(string(a))
}

type AppSecretValue string

func (a *AppSecretValue) Validate() error {
	if a == nil {
		return errors.New("secret should not be nil")
	}
	if len([]byte(*a)) < appSecretBytes {
		return errors.New("secret should be at least 32 bytes")
	}
	return nil
}

type AppSecret struct {
	Value   *AppSecretValue `yaml:"value" validate:"omitempty,validateFn"`
	Path    string          `yaml:"path" validate:"omitempty,filepath"`
	Version string          `yaml:"version"`
}

type Database struct {
	Port     uint16 `yaml:"port"`
	Host     string `yaml:"host" validate:"omitempty,hostname_rfc1123"`
	Database string `yaml:"database"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

type Fileserver struct {
	Volume    string `yaml:"volume" validate:"omitempty"`
	URLPrefix string `yaml:"url_prefix"`
}

type SMTP struct {
	TLSMode       TLSMode `yaml:"tls_mode" validate:"omitempty,validateFn"`
	Port          uint16  `yaml:"port"`
	TLSSkipVerify bool    `yaml:"tls_skip_verify"`
	Username      string  `yaml:"username"`
	Host          string  `yaml:"host" validate:"omitempty,hostname_rfc1123"`
	Password      string  `yaml:"password"`
	From          string  `yaml:"from" validate:"omitempty,email"`
}

type Admin struct {
	FirstName string        `yaml:"first_name" validate:"required_with_all=Email Password"`
	LastName  string        `yaml:"last_name" validate:"required_with_all=Email Password"`
	Email     string        `yaml:"email" validate:"omitempty,email"`
	Password  AdminPassword `yaml:"password" validate:"omitempty,validateFn"`
}

type Config struct {
	AppSecret  AppSecret  `yaml:"app_secret"`
	SMTP       SMTP       `yaml:"smtp"`
	Admin      Admin      `yaml:"admin"`
	Fileserver Fileserver `yaml:"fileserver"`
	Database   Database   `yaml:"database"`
	HostOrigin string     `yaml:"host_origin" validate:"url"`
	Env        string     `yaml:"env" validate:"omitempty,oneof=DEV PROD"`
}

func newAppSecret() (string, error) {
	token := make([]byte, appSecretBytes)
	if _, err := rand.Reader.Read(token); err != nil {
		return "", fmt.Errorf("creating app secret: %w", err)
	}
	return base64.StdEncoding.EncodeToString(token), nil
}

func loadAppSecret(config *Config) error {
	if config.AppSecret.Value != nil {
		return nil
	}

	var secret string
	if f1, err := os.Lstat(config.AppSecret.Path); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("checking secret path: %w", err)
		}

		file, err := os.OpenFile(config.AppSecret.Path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, appSecretFilePerms)
		if err != nil {
			return fmt.Errorf("creating secret file: %w", err)
		}
		defer func() { _ = file.Close() }()

		secret, err = newAppSecret()
		if err != nil {
			return fmt.Errorf("generating new app secret: %w", err)
		}

		if _, err := file.WriteString(secret); err != nil {
			return fmt.Errorf("writing secret file: %w", err)
		}
	} else {
		if f1.IsDir() {
			return fmt.Errorf("expected file, got directory at %q", config.AppSecret.Path)
		}
		data, err := os.ReadFile(config.AppSecret.Path)
		if err != nil {
			return fmt.Errorf("reading file: %w", err)
		}
		secret = string(data)
	}
	val := AppSecretValue(secret)
	config.AppSecret.Value = &val
	return nil
}

func loadWithDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func loadConfigFromEnv() (Config, error) {
	environment := loadWithDefault("ENV", EnvDev)
	hostOrigin := loadWithDefault("HOST_ORIGIN", "http://localhost:8080")

	// AppSecret
	appSecretValue := AppSecretValue(loadWithDefault("APP_SECRET", ""))
	appSecretPath := loadWithDefault("APP_SECRET_PATH", "/data/secret")
	appSecretVersion := loadWithDefault("APP_SECRET_VERSION", "1")

	// Database
	databasePort := loadWithDefault("DATABASE_PORT", "5432")
	databaseHost := loadWithDefault("DATABASE_HOST", "localhost")
	databaseDatabase := loadWithDefault("DATABASE", "")
	databaseUser := loadWithDefault("DATABASE_USER", "")
	databasePassword := loadWithDefault("DATABASE_PASSWORD", "")

	// Fileserver
	fileserverVolume := loadWithDefault("FILESERVER_VOLUME", "/data/files")
	fileserverURLPrefix := loadWithDefault("FILESERVER_URL_PREFIX", "/files")

	// SMTP
	smtpTLSMode := TLSMode(loadWithDefault("SMTP_TLS_MODE", string(TLSModeAuto)))
	smtpTLSSkipVerify := loadWithDefault("SMTP_TLS_SKIP_VERIFY", "false")
	smtpPort := loadWithDefault("SMTP_PORT", "587")
	smtpPassword := loadWithDefault("SMTP_PASSWORD", "")
	smtpFrom := loadWithDefault("SMTP_FROM", "")
	smtpUsername := loadWithDefault("SMTP_USERNAME", "")
	smtpHost := loadWithDefault("SMTP_HOST", "")

	// Admin
	adminFirstName := loadWithDefault("ADMIN_FIRST_NAME", "")
	adminLastName := loadWithDefault("ADMIN_LAST_NAME", "")
	adminEmail := loadWithDefault("ADMIN_EMAIL", "")
	adminPassword := AdminPassword(loadWithDefault("ADMIN_PASSWORD", ""))

	conf := Config{
		HostOrigin: hostOrigin,
		Env:        environment,
	}

	// Load App Secret
	conf.AppSecret = AppSecret{
		Path:    appSecretPath,
		Version: appSecretVersion,
	}
	if appSecretValue == "" {
		conf.AppSecret.Value = nil
	} else {
		conf.AppSecret.Value = &appSecretValue
	}

	// Load Database
	conf.Database = Database{
		Host:     databaseHost,
		Database: databaseDatabase,
		User:     databaseUser,
		Password: databasePassword,
	}
	if port, err := strconv.ParseUint(databasePort, 10, 16); err != nil {
		return conf, fmt.Errorf("invalid DATABASE_PORT (%q): %w", databaseDatabase, err)
	} else {
		conf.Database.Port = uint16(port)
	}

	// Load fileserver
	conf.Fileserver = Fileserver{
		Volume:    fileserverVolume,
		URLPrefix: fileserverURLPrefix,
	}

	// Load SMTP
	conf.SMTP = SMTP{
		Username: smtpUsername,
		Host:     smtpHost,
		Password: smtpPassword,
		From:     smtpFrom,
		TLSMode:  smtpTLSMode,
	}
	if b, err := strconv.ParseBool(smtpTLSSkipVerify); err != nil {
		return conf, fmt.Errorf("invalid TLS_SKIP_VERIFY (%q): %w", smtpTLSSkipVerify, err)
	} else {
		conf.SMTP.TLSSkipVerify = b
	}
	if port, err := strconv.ParseUint(smtpPort, 10, 16); err != nil {
		return conf, fmt.Errorf("invalid SMTP_PORT (%q): %w", smtpPort, err)
	} else {
		conf.SMTP.Port = uint16(port)
	}

	// Load Admin
	conf.Admin = Admin{
		FirstName: adminFirstName,
		LastName:  adminLastName,
		Email:     adminEmail,
		Password:  adminPassword,
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(conf); err != nil {
		return conf, err
	}

	if err := loadAppSecret(&conf); err != nil {
		return conf, fmt.Errorf("loading app secret: %w", err)
	}

	return conf, nil
}

func loadConfigFromFile(path string) (Config, error) {
	// Read file
	contents, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("reading config: %w", err)
	}

	// Unmarshal into config
	var config Config
	if err := yaml.Unmarshal(contents, &config); err != nil {
		return Config{}, fmt.Errorf("unmarshaling config: %w", err)
	}

	// Set defaults
	if config.AppSecret.Path == "" {
		config.AppSecret.Path = "/data/secret"
	}
	if config.AppSecret.Version == "" {
		config.AppSecret.Version = "1"
	}
	if config.Env == "" {
		config.Env = EnvDev
	}
	if config.HostOrigin == "" {
		config.HostOrigin = "http://localhost:8080"
	}
	if config.Database.Host == "" {
		config.Database.Host = "localhost"
	}
	if config.Database.Port == 0 {
		config.Database.Port = 5432
	}
	if config.Fileserver.Volume == "" {
		config.Fileserver.Volume = "/data/files"
	}
	if config.Fileserver.URLPrefix == "" {
		config.Fileserver.URLPrefix = "/files"
	}
	if config.SMTP.Port == 0 {
		config.SMTP.Port = 587
	}
	if config.SMTP.TLSMode == "" {
		config.SMTP.TLSMode = TLSModeAuto
	}

	// Validate config
	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(config); err != nil {
		return Config{}, err
	}

	if err := loadAppSecret(&config); err != nil {
		return Config{}, fmt.Errorf("loading app secret: %w", err)
	}

	return config, nil
}

func configFileExists(path string) bool {
	f, err := os.Lstat(path)
	if err != nil {
		return false
	}

	return !f.IsDir()
}

func LoadConfig() (Config, error) {
	if configFileExists(configFilePath) {
		return loadConfigFromFile(configFilePath)
	}

	return loadConfigFromEnv()
}
