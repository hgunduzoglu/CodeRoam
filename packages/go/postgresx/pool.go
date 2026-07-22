package postgresx

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	maxPoolConnections    int32 = 10
	maxConnectionLifetime       = 30 * time.Minute
	maxConnectionIdleTime       = 5 * time.Minute
	poolHealthCheckPeriod       = 30 * time.Second
	poolPingTimeout             = 5 * time.Second
)

var ErrInvalidPoolConfig = errors.New("invalid PostgreSQL pool configuration")

// OpenPool creates a bounded PostgreSQL pool and verifies connectivity before returning it.
func OpenPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, fmt.Errorf("%w: DSN is required", ErrInvalidPoolConfig)
	}

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("%w: DSN parse failed", ErrInvalidPoolConfig)
	}
	config.MaxConns = maxPoolConnections
	config.MaxConnLifetime = maxConnectionLifetime
	config.MaxConnIdleTime = maxConnectionIdleTime
	config.HealthCheckPeriod = poolHealthCheckPeriod
	config.PingTimeout = poolPingTimeout

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("open PostgreSQL pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping PostgreSQL pool: %w", err)
	}
	return pool, nil
}
