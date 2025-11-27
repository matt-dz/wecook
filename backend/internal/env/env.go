// Package env provides a structure for managing application-wide dependencies.
package env

import (
	"context"
	"log/slog"

	"github.com/matt-dz/wecook/internal/database"
	"github.com/matt-dz/wecook/internal/log"
)

type envKeyType struct{}

var envKey envKeyType

type Env struct {
	Log      *slog.Logger
	Database *database.Database
}

func New(logger *slog.Logger, database *database.Database) *Env {
	if logger == nil {
		logger = log.NullLogger()
	}

	return &Env{
		Log:      logger,
		Database: database,
	}
}

func Null() *Env {
	return &Env{
		Log:      log.NullLogger(),
		Database: nil,
	}
}

func EnvFromCtx(ctx context.Context) *Env {
	envValue := ctx.Value(envKey)
	if envValue == nil {
		return Null()
	}
	if env, ok := envValue.(*Env); ok {
		return env
	}

	return Null()
}

func WithCtx(ctx context.Context, env *Env) context.Context {
	return context.WithValue(ctx, envKey, env)
}
