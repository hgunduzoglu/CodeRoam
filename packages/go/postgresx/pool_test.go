package postgresx

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"
)

func TestOpenPoolRejectsMissingOrMalformedDSN(t *testing.T) {
	tests := map[string]string{
		"missing":    "",
		"whitespace": "   ",
		"malformed":  "://not-a-dsn",
	}
	for name, dsn := range tests {
		t.Run(name, func(t *testing.T) {
			pool, err := OpenPool(context.Background(), dsn)
			if pool != nil {
				pool.Close()
				t.Fatal("OpenPool() returned a pool for invalid configuration")
			}
			if !errors.Is(err, ErrInvalidPoolConfig) {
				t.Fatalf("OpenPool() error = %v, want ErrInvalidPoolConfig", err)
			}
		})
	}
}

func TestOpenPoolIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_TEST_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_TEST_DSN is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := OpenPool(ctx, dsn)
	if err != nil {
		t.Fatalf("OpenPool() error = %v", err)
	}
	defer pool.Close()

	config := pool.Config()
	if config.MaxConns != maxPoolConnections || config.MaxConnLifetime != maxConnectionLifetime ||
		config.MaxConnIdleTime != maxConnectionIdleTime || config.HealthCheckPeriod != poolHealthCheckPeriod ||
		config.PingTimeout != poolPingTimeout {
		t.Fatal("OpenPool() did not apply the bounded pool configuration")
	}
	var result int
	if err := pool.QueryRow(ctx, `SELECT 1`).Scan(&result); err != nil {
		t.Fatalf("query through pool: %v", err)
	}
	if result != 1 {
		t.Fatalf("query result = %d, want 1", result)
	}
}
