package database

import (
	"context"
	"fmt"
	"net"
	"net/url"

	"laundry-api/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(ctx context.Context) (*pgxpool.Pool, error) {
	connectionURL := url.URL{
		Scheme: "postgres",
		User: url.UserPassword(
			config.GetEnv("DB_USER", "postgres"),
			config.GetEnv("DB_PASSWORD", ""),
		),
		Host: net.JoinHostPort(
			config.GetEnv("DB_HOST", "localhost"),
			config.GetEnv("DB_PORT", "5432"),
		),
		Path: "/" + config.GetEnv("DB_NAME", "laundry_db"),
	}

	query := connectionURL.Query()
	query.Set("sslmode", config.GetEnv("DB_SSLMODE", "disable"))
	connectionURL.RawQuery = query.Encode()

	pool, err := pgxpool.New(ctx, connectionURL.String())
	if err != nil {
		return nil, fmt.Errorf("membuat connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("menghubungi database: %w", err)
	}

	return pool, nil
}