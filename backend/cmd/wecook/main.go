package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/mail"
	"os"
	"path/filepath"

	"github.com/matt-dz/wecook/internal/api"
	"github.com/matt-dz/wecook/internal/argon2id"
	"github.com/matt-dz/wecook/internal/database"
	"github.com/matt-dz/wecook/internal/env"
	"github.com/matt-dz/wecook/internal/fileserver"
	"github.com/matt-dz/wecook/internal/http"
	"github.com/matt-dz/wecook/internal/log"
	"github.com/matt-dz/wecook/internal/password"

	"github.com/jackc/pgx/v5/pgxpool"
)

func initDB(ctx context.Context, logger *slog.Logger) (*database.Database, error) {
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
	logger.Info("Connecting to database")
	pool, err := pgxpool.New(context.Background(), dbString)
	if err != nil {
		return nil, fmt.Errorf("creating database pool: %w", err)
	}

	db := database.NewDatabase(pool)

	// Ensuring database exists
	logger.Info("Ensuring database exists")
	if err := db.EnsureSchema(ctx); err != nil {
		return nil, fmt.Errorf("initializing database: %w", err)
	}

	return db, nil
}

func setupAdmin(env *env.Env, ctx context.Context) error {
	// Get email and password
	adminEmail, adminPassword := env.Get("ADMIN_EMAIL"), env.Get("ADMIN_PASSWORD")
	if adminEmail == "" || adminPassword == "" {
		env.Logger.Info("ADMIN_EMAIL and ADMIN_PASSWORD not setup, skipping admin setup")
		return nil
	}

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

func main() {
	env := env.New(log.New(nil), nil, http.New(), nil, nil)
	env.HTTP.Logger = env.Logger

	fileserverVolume := env.Get("FILESERVER_VOLUME")
	if fileserverVolume == "" {
		env.Logger.Error("environment variable FILESERVER_VOLUME not defined")
		os.Exit(1)
	}
	fileserverPath, err := filepath.Abs(fileserverVolume)
	if err != nil {
		env.Logger.Error("invalid FILESERVER_VOLUME value", slog.Any("error", err))
		os.Exit(1)
	}
	nginxURL := env.Get("NGINX_URL")
	if nginxURL == "" {
		env.Logger.Error("environment variable NGINX_URL not defined")
		os.Exit(1)
	}
	env.FileServer = fileserver.New(fileserverPath, nginxURL)

	db, err := initDB(context.TODO(), env.Logger)
	if err != nil {
		env.Logger.Error("Failed to initialize database", slog.Any("error", err))
		os.Exit(1)
	}
	env.Database = db
	if err := setupAdmin(env, context.Background()); err != nil {
		env.Logger.Warn("failed to setup admin", slog.Any("error", err))
		os.Exit(1)
	}

	if err := api.Start(env); err != nil {
		env.Logger.Error("API Failed", slog.Any("error", err))
		os.Exit(1)
	}
}
