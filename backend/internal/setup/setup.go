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
	"github.com/matt-dz/wecook/internal/argon2id"
	"github.com/matt-dz/wecook/internal/database"
	"github.com/matt-dz/wecook/internal/email"
	"github.com/matt-dz/wecook/internal/env"
	"github.com/matt-dz/wecook/internal/filestore"
	"github.com/matt-dz/wecook/internal/password"
)

// SMTP creates a new SMTP sender from environment variables.
// TLS usage is automatically inferred from the port:
// - Port 587 or 465: TLS enabled.
// - Other ports: TLS disabled.
func SMTP() (*email.SMTPSender, error) {
	host := os.Getenv("SMTP_HOST")
	if host == "" {
		return nil, fmt.Errorf("SMTP_HOST environment variable not set")
	}

	portStr := os.Getenv("SMTP_PORT")
	if portStr == "" {
		return nil, fmt.Errorf("SMTP_PORT environment variable not set")
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("invalid SMTP_PORT value: %w", err)
	}

	username := os.Getenv("SMTP_USERNAME")
	if username == "" {
		return nil, fmt.Errorf("SMTP_USERNAME environment variable not set")
	}

	password := os.Getenv("SMTP_PASSWORD")
	if password == "" {
		return nil, fmt.Errorf("SMTP_PASSWORD environment variable not set")
	}

	from := os.Getenv("SMTP_FROM")
	if from == "" {
		return nil, fmt.Errorf("SMTP_FROM environment variable not set")
	}

	config := email.Config{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
		From:     from,
	}

	return email.NewSMTPSender(config), nil
}

func Database() (*database.Database, error) {
	dbUser := os.Getenv("DATABASE_USER")
	if dbUser == "" {
		return nil, errors.New("environment variable DATABASE_USER must be set")
	}
	dbPassword := os.Getenv("DATABASE_PASSWORD")
	if dbPassword == "" {
		return nil, errors.New("environment variable DATABASE_PASSWORD must be set")
	}
	dbHost := os.Getenv("DATABASE_HOST")
	if dbHost == "" {
		return nil, errors.New("environment variable DATABASE_HOST must be set")
	}
	dbPort := os.Getenv("DATABASE_PORT")
	if dbPort == "" {
		return nil, errors.New("environment variable DATABASE_PORT must be set")
	}
	defaultDB := os.Getenv("DATABASE")
	if defaultDB == "" {
		return nil, errors.New("environment variable DATABASE must be set")
	}
	dbString := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s", dbUser, dbPassword, dbHost, dbPort, defaultDB)

	// Creating DB connection
	pool, err := pgxpool.New(context.Background(), dbString)
	if err != nil {
		return nil, fmt.Errorf("creating database pool: %w", err)
	}

	db := database.NewDatabase(pool)
	if err := db.EnsureSchema(context.TODO()); err != nil {
		return nil, fmt.Errorf("initializing database: %w", err)
	}

	return db, nil
}

// Admin setups an admin user if one does not exist. Requires env.Database.
func Admin(env *env.Env) error {
	adminEmail, adminPassword := env.Get("ADMIN_EMAIL"), env.Get("ADMIN_PASSWORD")
	if adminEmail == "" || adminPassword == "" {
		env.Logger.Info("ADMIN_EMAIL and ADMIN_PASSWORD not setup, skipping admin setup")
		return nil
	}

	ctx := context.TODO()

	// Validate email and password
	if _, err := mail.ParseAddress(adminEmail); err != nil {
		return fmt.Errorf("parsing admin email: %w", err)
	}
	if err := password.ValidatePassword(adminPassword); err != nil {
		return fmt.Errorf("validating admin password: %w", err)
	}

	// Check admin count
	count, err := env.Database.GetAdminCount(ctx)
	if err != nil {
		return fmt.Errorf("getting admin count: %w", err)
	}
	if count > 0 {
		env.Logger.Info("admin already setup, skipping setup")
		return nil
	}

	hashedPassword, err := argon2id.EncodeHash(adminPassword, argon2id.DefaultParams)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}

	// Create admin
	_, err = env.Database.CreateAdmin(ctx, database.CreateAdminParams{
		FirstName:    "admin",
		LastName:     "admin",
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
		return fs, errors.New("environment variable FILESERVER_VOLUME not defined")
	}
	fileserverPath, err := filepath.Abs(fileserverVolume)
	if err != nil {
		return fs, fmt.Errorf("creating fileserver path: %w", err)
	}
	urlPrefix := os.Getenv("FILESERVER_URL_PREFIX")
	if urlPrefix == "" {
		urlPrefix = filestore.DefaultURLPrefix
	}
	filestoreHost := os.Getenv("NGINX_URL")
	if filestoreHost == "" {
		return fs, errors.New("NGINX_URL must be set")
	}
	return filestore.New(fileserverPath, urlPrefix, filestoreHost), nil
}
