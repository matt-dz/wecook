// Package env provides a structure for managing application-wide dependencies.
package env

import (
	"context"
	"log/slog"
	"os"

	"github.com/matt-dz/wecook/internal/database"
	"github.com/matt-dz/wecook/internal/filestore"
	"github.com/matt-dz/wecook/internal/http"
	"github.com/matt-dz/wecook/internal/log"
)

type envKeyType struct{}

var envKey envKeyType

type Env struct {
	Logger    *slog.Logger
	Database  *database.Database
	HTTP      *http.HTTP
	FileStore filestore.FileStoreInterface
	vars      map[string]string
}

func (e *Env) Get(key string) string {
	if v, found := e.vars[key]; found {
		return v
	}
	return os.Getenv(key)
}

func (e *Env) IsProd() bool {
	return e.Get("ENV") == "PROD"
}

func New(logger *slog.Logger, database *database.Database,
	http *http.HTTP, filestore filestore.FileStoreInterface,
	vars map[string]string,
) *Env {
	if logger == nil {
		logger = log.NullLogger()
	}

	return &Env{
		Logger:    logger,
		Database:  database,
		HTTP:      http,
		FileStore: filestore,
		vars:      vars,
	}
}

func Null() *Env {
	return &Env{
		Logger:    log.NullLogger(),
		Database:  nil,
		HTTP:      nil,
		FileStore: nil,
		vars:      make(map[string]string),
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
