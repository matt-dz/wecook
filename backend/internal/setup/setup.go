// Package setup is responsible for setting up components.
package setup

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/matt-dz/wecook/internal/argon2id"
	"github.com/matt-dz/wecook/internal/config"
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
func SMTP(config config.Config) (*email.SMTPSender, error) {
	emailConfig := email.Config{
		Host:                config.SMTP.Host,
		Port:                int(config.SMTP.Port),
		Username:            config.SMTP.Username,
		Password:            config.SMTP.Password,
		From:                config.SMTP.From,
		TLSMode:             email.TLSMode(config.SMTP.TLSMode),
		SkipTLSVerification: config.SMTP.TLSSkipVerify,
	}

	return email.NewSMTPSender(emailConfig), nil
}

func Database(ctx context.Context, config config.Config) (*database.Database, error) {
	poolConfig, err := pgxpool.ParseConfig("")
	if err != nil {
		return nil, fmt.Errorf("configuring database pool: %w", err)
	}

	poolConfig.ConnConfig.Host = config.Database.Host
	poolConfig.ConnConfig.Port = config.Database.Port
	poolConfig.ConnConfig.User = config.Database.User
	poolConfig.ConnConfig.Password = config.Database.Password
	poolConfig.ConnConfig.Database = config.Database.Database

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

func FileStore(config config.Config) (filestore.FileStore, error) {
	return filestore.New(config.Fileserver.Volume, config.Fileserver.URLPrefix, config.HostOrigin), nil
}

func Preferences(ctx context.Context, env *env.Env, id int32) error {
	return env.Database.CreatePreferences(ctx, id)
}
