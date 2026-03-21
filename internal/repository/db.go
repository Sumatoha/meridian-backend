package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DBTX is the interface for database transactions.
type DBTX interface {
	Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error)
	Query(context.Context, string, ...interface{}) (pgx.Rows, error)
	QueryRow(context.Context, string, ...interface{}) pgx.Row
}

// Queries provides all database query methods.
type Queries struct {
	db DBTX
}

// New creates a new Queries instance.
func New(db DBTX) *Queries {
	return &Queries{db: db}
}

// WithTx creates a new Queries instance bound to a transaction.
func (q *Queries) WithTx(tx pgx.Tx) *Queries {
	return &Queries{db: tx}
}

// NewFromPool creates Queries from a pgxpool.Pool (which implements DBTX).
func NewFromPool(pool *pgxpool.Pool) *Queries {
	return &Queries{db: pool}
}
