package main

import (
	"context"
	"errors"
	"testing"

	"github.com/hgunduzoglu/coderoam/packages/go/postgresx"
)

func TestRunRequiresPostgresDSN(t *testing.T) {
	t.Setenv("POSTGRES_DSN", "")
	if err := run(context.Background()); !errors.Is(err, postgresx.ErrInvalidPoolConfig) {
		t.Fatalf("run() error = %v, want ErrInvalidPoolConfig", err)
	}
}
