// Package db provides utilities for managing PostgreSQL database
// connections and connection pools.
package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPool creates a configured PostgreSQL connection pool
func NewPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse DSN: %w", err)
	}

	config.MaxConns = 25                      // максимум 25 соединений
	config.MinConns = 2                       // минимум 2 всегда активны
	config.MaxConnLifetime = 30 * time.Minute // соединение живёт 30 минут
	config.MaxConnIdleTime = 5 * time.Minute  // простоит 5 мин – закрывается

	config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeCacheStatement

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return pool, nil
}
