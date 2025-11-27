package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/matt-dz/wecook/internal/api"
	"github.com/matt-dz/wecook/internal/database"
	"github.com/matt-dz/wecook/internal/env"
	"github.com/matt-dz/wecook/internal/log"

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
	if err := database.EnsureSchema(db, ctx); err != nil {
		return nil, fmt.Errorf("initializing database: %w", err)
	}

	return db, nil
}

func main() {
	logger := log.New(nil)

	db, err := initDB(context.TODO(), logger)
	if err != nil {
		logger.Error("Failed to initialize database", slog.Any("error", err))
	}

	env := env.New(logger, db)
	if err := api.Start(env); err != nil {
		logger.Error("API Failed", slog.Any("error", err))
		os.Exit(1)
	}
}
