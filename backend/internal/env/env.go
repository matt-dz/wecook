// Package env provides a structure for managing application-wide dependencies.
package env

import (
	"context"
	"log/slog"
	"os"

	"github.com/matt-dz/wecook/internal/database"
	mGarage "github.com/matt-dz/wecook/internal/garage"
	"github.com/matt-dz/wecook/internal/http"
	"github.com/matt-dz/wecook/internal/log"
)

type envKeyType struct{}

var envKey envKeyType

type Env struct {
	Logger   *slog.Logger
	Database *database.Database
	HTTP     *http.HTTP
	S3       *mGarage.APIClient
}

func (e *Env) Get(key string) string {
	return os.Getenv(key)
}

func New(logger *slog.Logger, database *database.Database, http *http.HTTP, S3 *mGarage.APIClient) *Env {
	if logger == nil {
		logger = log.NullLogger()
	}

	return &Env{
		Logger:   logger,
		Database: database,
		HTTP:     http,
		S3:       S3,
	}
}

func Null() *Env {
	return &Env{
		Logger:   log.NullLogger(),
		Database: nil,
		HTTP:     nil,
		S3:       nil,
	}
}

func EnvFromCtx(ctx context.Context) *Env {
	if env, ok := ctx.Value(envKey).(*Env); ok {
		return env
	}

	return Null()
}

func WithCtx(ctx context.Context, env *Env) context.Context {
	return context.WithValue(ctx, envKey, env)
}
