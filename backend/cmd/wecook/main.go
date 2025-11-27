package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/matt-dz/wecook/internal/api"
	"github.com/matt-dz/wecook/internal/database"
	"github.com/matt-dz/wecook/internal/env"
	"github.com/matt-dz/wecook/internal/log"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	logger := log.New(nil)

	dbUser := os.Getenv("DATABASE_USER")
	if dbUser == "" {
		logger.Error("environment variable DATABASE_USER must be set.")
		os.Exit(1)
	}
	dbPassword := os.Getenv("DATABASE_PASSWORD")
	if dbPassword == "" {
		logger.Error("environment variable DATABASE_PASSWORD must be set.")
		os.Exit(1)
	}
	dbHost := os.Getenv("DATABASE_HOST")
	if dbHost == "" {
		logger.Error("environment variable DATABASE_HOST must be set.")
		os.Exit(1)
	}
	dbPort := os.Getenv("DATABASE_PORT")
	if dbPort == "" {
		logger.Error("environment variable DATABASE_PORT must be set.")
		os.Exit(1)
	}
	db := os.Getenv("DATABASE")
	if db == "" {
		logger.Error("environment variable DATABASE must be set.")
		os.Exit(1)
	}
	dbString := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s", dbUser, dbPassword, dbHost, dbPort, db)

	pool, err := pgxpool.New(context.Background(), dbString)
	if err != nil {
		logger.Error("Failed to create database pool", slog.Any("error", err))
		os.Exit(1)
	}

	env := env.New(logger, database.NewDatabase(pool))
	if err := api.Start(env); err != nil {
		logger.Error("API Failed", slog.Any("error", err))
		os.Exit(1)
	}
}
