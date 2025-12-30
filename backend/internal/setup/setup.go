// Package setup is responsible for setting up components.
package setup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/matt-dz/wecook/internal/argon2id"
	"github.com/matt-dz/wecook/internal/database"
	"github.com/matt-dz/wecook/internal/email"
	"github.com/matt-dz/wecook/internal/env"
	"github.com/matt-dz/wecook/internal/filestore"
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
	adminEmail, adminPassword := env.Config.Admin.Email, string(env.Config.Admin.Password)
	if adminEmail == "" || adminPassword == "" {
		env.Logger.Info("ADMIN_EMAIL and ADMIN_PASSWORD not setup, skipping admin setup")
		return nil
	}

	// Hash password
	hashedPassword, err := argon2id.EncodeHash(adminPassword, argon2id.DefaultParams)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}

	// Create admin
	_, err = env.Database.CreateAdmin(ctx, database.CreateAdminParams{
		FirstName:    env.Config.Admin.FirstName,
		LastName:     env.Config.Admin.LastName,
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
