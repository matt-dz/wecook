// Package env provides a structure for managing application-wide dependencies.
package env

import (
	"log/slog"

	"github.com/matt-dz/wecook/internal/database"
	"github.com/matt-dz/wecook/internal/log"
)

type Env struct {
	Log      *slog.Logger
	Database *database.Database
}

func New(lg *slog.Logger, database *database.Database) *Env {
	if lg == nil {
		lg = slog.New(log.NullLog())
	}

	return &Env{
		Log:      lg,
		Database: database,
	}
}

func Null() *Env {
	return &Env{
		Log:      slog.New(log.NullLog()),
		Database: nil,
	}
}
