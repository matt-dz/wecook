// Package database
package database

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Pool interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

type Database struct {
	*Queries

	Pool Pool
}

func NewDatabase(pool *pgxpool.Pool) *Database {
	return &Database{
		Queries: New(pool),
		Pool:    pool,
	}
}
