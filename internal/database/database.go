package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type DB struct {
	conn *pgx.Conn
}

func New(ctx context.Context, connectionString string) (*DB, error) {
	conn, err := pgx.Connect(ctx, connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := conn.Ping(ctx); err != nil {
		conn.Close(ctx)
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{conn: conn}, nil
}

func (db *DB) Close(ctx context.Context) error {
	return db.conn.Close(ctx)
}
