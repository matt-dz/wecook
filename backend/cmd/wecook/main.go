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

	httpConfig := http.DefaultConfig()
	httpConfig.Logger = logger
	http := http.New(httpConfig)

	conf, err := config.LoadConfig()
	if err != nil {
		logger.Error("failed to load config", slog.Any("error", err))
		os.Exit(1)
	}

	fs, err := setup.FileStore(conf)
	if err != nil {
		logger.Error("failed to setup file store", slog.Any("error", err))
		os.Exit(1)
	}

	db, err := setup.Database(setupCtx, conf)
	if err != nil {
		logger.Error("failed to setup database", slog.Any("error", err))
		os.Exit(1)
	}

	smtpSender, err := setup.SMTP(conf)
	if err != nil {
		logger.Error("failed to setup SMTP sender", slog.Any("error", err))
		os.Exit(1)
	}

	env := &env.Env{
		Logger:    logger,
		FileStore: fs,
		Database:  db,
		SMTP:      smtpSender,
		HTTP:      http,
		Config:    conf,
	}

	logger.DebugContext(ctx, "setting up admin")
	if err := setup.Admin(setupCtx, env); err != nil {
		logger.Error("failed to setup admin", slog.Any("error", err))
		os.Exit(1)
	}

	logger.DebugContext(ctx, "setting up preferences")
	if err := setup.Preferences(setupCtx, env, config.PreferenceID); err != nil {
		logger.Error("failed to setup preferences", slog.Any("error", err))
		os.Exit(1)
	}

	if err := api.Start(env); err != nil {
		env.Logger.Error("API Failed", slog.Any("error", err))
		os.Exit(1)
	}
}
