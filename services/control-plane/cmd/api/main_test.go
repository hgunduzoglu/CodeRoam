package main

import (
	"context"
	"strings"
	"testing"
)

func TestRunRequiresPostgresDSN(t *testing.T) {
	t.Setenv("POSTGRES_DSN", "")
	if err := run(context.Background()); err == nil || !strings.Contains(err.Error(), "POSTGRES_DSN") {
		t.Fatalf("run() error = %v, want missing POSTGRES_DSN", err)
	}
}
