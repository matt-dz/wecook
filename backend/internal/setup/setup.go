// Package setup is responsible for setting up components.
package setup

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"os"
	"path/filepath"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/matt-dz/wecook/internal/api/token"
	"github.com/matt-dz/wecook/internal/argon2id"
	"github.com/matt-dz/wecook/internal/database"
	"github.com/matt-dz/wecook/internal/email"
	"github.com/matt-dz/wecook/internal/env"
	"github.com/matt-dz/wecook/internal/filestore"
	"github.com/matt-dz/wecook/internal/password"
)

const (
	appSecretPath = "/data/secret"
	secretPerms   = 0o600
)

// SMTP creates a new SMTP sender from environment variables.
// TLS usage is automatically inferred from the port unless overridden:
// - Port 587: StartTLS is used.
// - Port 465: Implicit TLS is used.
// - Other ports: TLS is disabled.
func SMTP() (*email.SMTPSender, error) {
	host := os.Getenv("SMTP_HOST")
	if host == "" {
		return nil, NewEnvironmentVariableMissingError("SMTP_HOST")
	}

	portStr := os.Getenv("SMTP_PORT")
	if portStr == "" {
		return nil, NewEnvironmentVariableMissingError("SMTP_PORT")
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("invalid SMTP_PORT value: %w", err)
	}

	username := os.Getenv("SMTP_USERNAME")
	if username == "" {
		return nil, NewEnvironmentVariableMissingError("SMTP_PORT")
	}

	password := os.Getenv("SMTP_PASSWORD")
	if password == "" {
		return nil, NewEnvironmentVariableMissingError("SMTP_PASSWORD")
	}

	from := os.Getenv("SMTP_FROM")
	if from == "" {
		return nil, NewEnvironmentVariableMissingError("SMTP_FROM")
	}

	tlsMode, err := email.ParseTLSMode(os.Getenv("SMTP_TLS_MODE"))
	if err != nil {
		return nil, err
	}

	skipVerify := false
	if skipVerifyStr := os.Getenv("SMTP_TLS_SKIP_VERIFY"); skipVerifyStr != "" {
		skipVerify, err = strconv.ParseBool(skipVerifyStr)
		if err != nil {
			return nil, fmt.Errorf("invalid SMTP_TLS_SKIP_VERIFY value: %w", err)
		}
	}

	config := email.Config{
		Host:                host,
		Port:                port,
		Username:            username,
		Password:            password,
		From:                from,
		TLSMode:             tlsMode,
		SkipTLSVerification: skipVerify,
	}

	return email.NewSMTPSender(config), nil
}

func Database(ctx context.Context) (*database.Database, error) {
	dbUser := os.Getenv("DATABASE_USER")
	if dbUser == "" {
		return nil, NewEnvironmentVariableMissingError("DATABASE_USER")
	}
	dbPassword := os.Getenv("DATABASE_PASSWORD")
	if dbPassword == "" {
		return nil, NewEnvironmentVariableMissingError("DATABASE_PASSWORD")
	}
	dbHost := os.Getenv("DATABASE_HOST")
	if dbHost == "" {
		return nil, NewEnvironmentVariableMissingError("DATABASE_HOST")
	}
	dbPort := os.Getenv("DATABASE_PORT")
	if dbPort == "" {
		return nil, NewEnvironmentVariableMissingError("DATABASE_PORT")
	}
	defaultDB := os.Getenv("DATABASE")
	if defaultDB == "" {
		return nil, NewEnvironmentVariableMissingError("DATABASE")
	}

	poolConfig, err := pgxpool.ParseConfig("")
	if err != nil {
		return nil, fmt.Errorf("configuring database pool: %w", err)
	}

	port, err := strconv.ParseUint(dbPort, 10, 16)
	if err != nil {
		return nil, fmt.Errorf("parsing DATABASE_PORT: %w", err)
	}

	poolConfig.ConnConfig.Host = dbHost
	poolConfig.ConnConfig.Port = uint16(port)
	poolConfig.ConnConfig.User = dbUser
	poolConfig.ConnConfig.Password = dbPassword
	poolConfig.ConnConfig.Database = defaultDB

	// Creating DB connection
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("creating database pool: %w", err)
	}

	db := database.NewDatabase(pool)
	if err := db.EnsureSchema(ctx); err != nil {
		return nil, fmt.Errorf("initializing database: %w", err)
	}

	return db, nil
}

// Admin setups an admin user if one does not exist. Requires env.Database.
func Admin(ctx context.Context, env *env.Env) error {
	// Check admin count
	count, err := env.Database.GetAdminCount(ctx)
	if err != nil {
		return fmt.Errorf("getting admin count: %w", err)
	}
	if count > 0 {
		env.Logger.Info("admin already setup, skipping setup")
		return nil
	}

	// Get admin info
	adminEmail, adminPassword := env.Get("ADMIN_EMAIL"), env.Get("ADMIN_PASSWORD")
	if adminEmail == "" || adminPassword == "" {
		env.Logger.Info("ADMIN_EMAIL and ADMIN_PASSWORD not setup, skipping admin setup")
		return nil
	}
	adminFirstName := env.Get("ADMIN_FIRST_NAME")
	if adminFirstName == "" {
		return NewEnvironmentVariableMissingError("ADMIN_FIRST_NAME")
	}
	adminLastName := env.Get("ADMIN_LAST_NAME")
	if adminLastName == "" {
		return NewEnvironmentVariableMissingError("ADMIN_LAST_NAME")
	}

	// Validate email and password
	if _, err := mail.ParseAddress(adminEmail); err != nil {
		return fmt.Errorf("parsing admin email: %w", err)
	}
	if err := password.ValidatePassword(adminPassword); err != nil {
		return fmt.Errorf("validating admin password: %w", err)
	}

	hashedPassword, err := argon2id.EncodeHash(adminPassword, argon2id.DefaultParams)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}

	// Create admin
	_, err = env.Database.CreateAdmin(ctx, database.CreateAdminParams{
		FirstName:    adminFirstName,
		LastName:     adminLastName,
		PasswordHash: hashedPassword,
		Email:        adminEmail,
	})
	if err != nil {
		return fmt.Errorf("creating admin: %w", err)
	}
	env.Logger.Info("successfully setup admin!")

	return nil
}

func FileStore() (filestore.FileStore, error) {
	var fs filestore.FileStore
	fileserverVolume := os.Getenv("FILESERVER_VOLUME")
	if fileserverVolume == "" {
		return fs, NewEnvironmentVariableMissingError("FILESERVER_VOLUME")
	}
	fileserverPath, err := filepath.Abs(fileserverVolume)
	if err != nil {
		return fs, fmt.Errorf("creating fileserver path: %w", err)
	}
	urlPrefix := os.Getenv("FILESERVER_URL_PREFIX")
	if urlPrefix == "" {
		urlPrefix = filestore.DefaultURLPrefix
	}
	filestoreHost := os.Getenv("HOST_ORIGIN")
	if filestoreHost == "" {
		return fs, NewEnvironmentVariableMissingError("HOST_ORIGIN")
	}
	return filestore.New(fileserverPath, urlPrefix, filestoreHost), nil
}

func AppSecret(env *env.Env) error {
	var secretPath string
	if env.Get("APP_SECRET_PATH") != "" {
		secretPath = env.Get("APP_SECRET_PATH")
	} else {
		secretPath = appSecretPath
	}

	if env.Get("APP_SECRET") != "" {
		return nil
	}

	var secret string
	if f1, err := os.Lstat(secretPath); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("checking secret path: %w", err)
		}

		file, err := os.OpenFile(secretPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, secretPerms)
		if err != nil {
			return fmt.Errorf("creating secret file: %w", err)
		}
		defer func() { _ = file.Close() }()

		secret, err = token.NewAppSecret()
		if err != nil {
			return fmt.Errorf("generating new app secret: %w", err)
		}

		if _, err := file.WriteString(secret); err != nil {
			return fmt.Errorf("writing secret file: %w", err)
		}
	} else {
		if f1.IsDir() {
			return fmt.Errorf("expected file, got directory at %q", secretPath)
		}
		data, err := os.ReadFile(secretPath)
		if err != nil {
			return fmt.Errorf("reading file: %w", err)
		}
		secret = string(data)
	}

	env.Set("APP_SECRET", secret)
	return nil
}
