// Package config contains utilities for loading configs
package config

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

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

func splitFieldList(param string) []string {
	// "A,B,C" or "A B C"
	param = strings.ReplaceAll(param, " ", ",")
	parts := strings.Split(param, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// allOrNothing implements a cross-field validator for go-playground/validator.
//
// It enforces an “all-or-nothing” rule across a set of fields specified in the
// validation tag parameters. The validator succeeds only if either:
//
//  1. All listed fields have zero values, or
//  2. All listed fields have non-zero values.
//
// Any mixed state—where at least one field is zero-valued and at least one field
// is non-zero—causes validation to fail.
//
// The validator must be attached to a placeholder field and inspects the parent
// struct to perform validation. Field names are provided as a comma- or
// space-separated list via the tag parameter (e.g. `validate:"allornothing=A,B,C"`).
//
// Pointer and interface fields are handled as follows:
//   - A nil pointer or interface is treated as a zero value.
//   - A non-nil pointer or interface is dereferenced until a concrete value is
//     reached, and that value is evaluated using reflect.Value.IsZero.
//
// If the parent value is nil, the parent is not a struct, a referenced field
// does not exist, or no field names are provided, the validation fails to signal
// misconfiguration.
//
// This validator is intended for enforcing atomic field groups in API inputs
// (e.g. ensuring related fields are either all provided or all omitted).
func allOrNothing(fl validator.FieldLevel) bool {
	parent := fl.Parent()
	if parent.Kind() == reflect.Pointer {
		if parent.IsNil() {
			return true // nothing to validate
		}
		parent = parent.Elem()
	}
	if parent.Kind() != reflect.Struct {
		return false
	}

	names := splitFieldList(fl.Param())
	if len(names) == 0 {
		return false
	}

	hasZero := false
	hasNonZero := false

	for _, name := range names {
		f := parent.FieldByName(name)
		if !f.IsValid() {
			return false // field name typo / not found
		}

		// Treat pointers/interfaces as zero if nil, otherwise unwrap
		for (f.Kind() == reflect.Pointer || f.Kind() == reflect.Interface) && !f.IsNil() {
			f = f.Elem()
		}

		if f.IsZero() {
			hasZero = true
		} else {
			hasNonZero = true
		}

		// Mixed state detected → invalid
		if hasZero && hasNonZero {
			return false
		}
	}

	return true
}

func registerAllOrNothing(v *validator.Validate) {
	_ = v.RegisterValidation("allOrNothing", allOrNothing)
}

func formatValidationError(err error) error {
	validationErrs, ok := err.(validator.ValidationErrors) //nolint:errorlint
	if !ok {
		return err
	}

	for _, e := range validationErrs {
		if e.Tag() == "allOrNothing" {
			// Extract the struct name from the namespace
			// e.g., "Config.SMTP.Validate" -> "SMTP"
			namespace := e.Namespace()
			parts := strings.Split(namespace, ".")
			var structName string
			//nolint:mnd
			if len(parts) >= 2 {
				structName = parts[len(parts)-2]
			}

			var fields string
			switch structName {
			case "SMTP":
				fields = "From, Password, Host, Username, and Port"
			case "Database":
				fields = "Port, Host, Database, User, and Password"
			case "Admin":
				fields = "FirstName, LastName, Email, and Password"
			default:
				fields = "all related fields"
			}

			return fmt.Errorf(
				"%s configuration is incomplete: either all fields must be set (%s) or all must be empty",
				structName, fields)
		}
	}

	return err
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

	Validate struct{} `yaml:"-" validate:"allOrNothing=Port Host Database User Password"`
}

type Fileserver struct {
	Volume    string `yaml:"volume"`
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

	Validate struct{} `yaml:"-" validate:"allOrNothing=From Password Host Username Port"`
}

type Admin struct {
	FirstName string        `yaml:"first_name" validate:"required_with_all=Email Password"`
	LastName  string        `yaml:"last_name" validate:"required_with_all=Email Password"`
	Email     string        `yaml:"email" validate:"omitempty,email"`
	Password  AdminPassword `yaml:"password" validate:"omitempty,validateFn"`

	Validate struct{} `yaml:"-" validate:"allOrNothing=FirstName LastName Email Password"`
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
	smtpPassword := loadWithDefault("SMTP_PASSWORD", "")
	smtpFrom := loadWithDefault("SMTP_FROM", "")
	smtpUsername := loadWithDefault("SMTP_USERNAME", "")
	smtpHost := loadWithDefault("SMTP_HOST", "")

	// Only set SMTP_PORT default if SMTP is being configured
	smtpPort := loadWithDefault("SMTP_PORT", "")
	if smtpPort == "" && (smtpFrom != "" || smtpPassword != "" || smtpHost != "" || smtpUsername != "") {
		smtpPort = "587"
	}

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
	if smtpPort != "" {
		if port, err := strconv.ParseUint(smtpPort, 10, 16); err != nil {
			return conf, fmt.Errorf("invalid SMTP_PORT (%q): %w", smtpPort, err)
		} else {
			conf.SMTP.Port = uint16(port)
		}
	}

	// Load Admin
	conf.Admin = Admin{
		FirstName: adminFirstName,
		LastName:  adminLastName,
		Email:     adminEmail,
		Password:  adminPassword,
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	registerAllOrNothing(validate)
	if err := validate.Struct(conf); err != nil {
		return conf, formatValidationError(err)
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
	// Only set SMTP.Port default if SMTP is being configured
	if config.SMTP.Port == 0 && (config.SMTP.From != "" || config.SMTP.Password != "" ||
		config.SMTP.Host != "" || config.SMTP.Username != "") {
		config.SMTP.Port = 587
	}
	if config.SMTP.TLSMode == "" {
		config.SMTP.TLSMode = TLSModeAuto
	}

	// Validate config
	validate := validator.New(validator.WithRequiredStructEnabled())
	registerAllOrNothing(validate)
	if err := validate.Struct(config); err != nil {
		return Config{}, formatValidationError(err)
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
