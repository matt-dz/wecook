// Package env provides a structure for managing application-wide dependencies.
package env

import (
	"context"
	"log/slog"
	"os"

	"github.com/matt-dz/wecook/internal/config"
	"github.com/matt-dz/wecook/internal/database"
	"github.com/matt-dz/wecook/internal/email"
	"github.com/matt-dz/wecook/internal/filestore"
	"github.com/matt-dz/wecook/internal/http"
	"github.com/matt-dz/wecook/internal/log"
)

type envKeyType struct{}

var envKey envKeyType

type Env struct {
	Logger    *slog.Logger
	Database  database.Querier
	HTTP      http.HTTPDoer
	SMTP      email.Sender
	FileStore filestore.FileStoreInterface
	Config    config.Config
	vars      map[string]string
}

func (e *Env) Get(key string) string {
	if v, found := e.vars[key]; found {
		return v
	}
	return os.Getenv(key)
}

func (e *Env) Set(key string, value string) {
	if e.vars == nil {
		e.vars = make(map[string]string)
	}
	e.vars[key] = value
}

func (e *Env) IsProd() bool {
	return e.Config.Env == config.EnvProd
}

func New(vars map[string]string) *Env {
	return &Env{
		vars: vars,
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
