// Package database
package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/matt-dz/wecook/internal/sql"
)

type Pool interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

type Database struct {
	Querier

	db DBTX
}

func NewDatabase(db DBTX) *Database {
	return &Database{
		Querier: New(db),
		db:      db,
	}
}

// EnsureSchema ensures the database schema is applied to the
// Postgres database. The schema is applied to the database
// if the schema is not detected.
func (db *Database) EnsureSchema(ctx context.Context) error {
	exists, err := db.CheckUsersTableExists(ctx)
	if err != nil {
		return fmt.Errorf("ensuring schema exists: %w", err)
	}

	if exists {
		return nil
	}

	if _, err := db.db.Exec(ctx, sql.Schema()); err != nil {
		return fmt.Errorf("applying database schema: %w", err)
	}

	return nil
}
