package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/matt-dz/wecook/internal/api"
	"github.com/matt-dz/wecook/internal/config"
	"github.com/matt-dz/wecook/internal/env"
	"github.com/matt-dz/wecook/internal/http"
	"github.com/matt-dz/wecook/internal/log"
	"github.com/matt-dz/wecook/internal/setup"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	const setupTime = 30 * time.Second
	setupCtx, cancel := context.WithTimeout(ctx, setupTime)
	defer cancel()

	logger := log.New(nil)

	conf := http.DefaultConfig()
	conf.Logger = logger
	http := http.New(conf)

	fs, err := setup.FileStore()
	if err != nil {
		logger.Error("failed to setup file store", slog.Any("error", err))
		os.Exit(1)
	}

	db, err := setup.Database(setupCtx)
	if err != nil {
		logger.Error("failed to setup database", slog.Any("error", err))
		os.Exit(1)
	}

	smtpSender, err := setup.SMTP()
	if err != nil {
		logger.Error("failed to setup SMTP sender", slog.Any("error", err))
		os.Exit(1)
	}

	config, err := config.LoadConfig()
	if err != nil {
		logger.Error("failed to load config", slog.Any("error", err))
		os.Exit(1)
	}

	env := &env.Env{
		Logger:    logger,
		FileStore: fs,
		Database:  db,
		SMTP:      smtpSender,
		HTTP:      http,
		Config:    config,
	}

	if err := setup.Admin(setupCtx, env); err != nil {
		logger.Error("failed to setup admin", slog.Any("error", err))
		os.Exit(1)
	}

	if err := api.Start(env); err != nil {
		env.Logger.Error("API Failed", slog.Any("error", err))
		os.Exit(1)
	}
}
