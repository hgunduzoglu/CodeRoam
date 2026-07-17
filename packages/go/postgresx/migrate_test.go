package postgresx

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type failingTransactionStarter struct {
	called bool
}

func (starter *failingTransactionStarter) Begin(context.Context) (pgx.Tx, error) {
	starter.called = true
	return nil, errors.New("unexpected transaction")
}

func TestApplyMigrationsRejectsInvalidDefinitionsBeforeOpeningTransaction(t *testing.T) {
	tests := map[string][]Migration{
		"empty scope":  {{Version: 1, Name: "init", SQL: "SELECT 1"}},
		"padded scope": {{Scope: " auth", Version: 1, Name: "init", SQL: "SELECT 1"}},
		"zero version": {{Scope: "auth", Name: "init", SQL: "SELECT 1"}},
		"empty name":   {{Scope: "auth", Version: 1, SQL: "SELECT 1"}},
		"padded name":  {{Scope: "auth", Version: 1, Name: " init", SQL: "SELECT 1"}},
		"empty SQL":    {{Scope: "auth", Version: 1, Name: "init"}},
		"duplicate version": {
			{Scope: "auth", Version: 1, Name: "init", SQL: "SELECT 1"},
			{Scope: "auth", Version: 1, Name: "duplicate", SQL: "SELECT 2"},
		},
	}

	for name, migrations := range tests {
		t.Run(name, func(t *testing.T) {
			starter := &failingTransactionStarter{}
			err := ApplyMigrations(context.Background(), starter, migrations)
			if !errors.Is(err, ErrInvalidMigration) {
				t.Fatalf("ApplyMigrations() error = %v, want ErrInvalidMigration", err)
			}
			if starter.called {
				t.Fatal("ApplyMigrations() opened a transaction for invalid input")
			}
		})
	}
}

func TestApplyMigrationsIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_TEST_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_TEST_DSN is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect to test PostgreSQL: %v", err)
	}
	t.Cleanup(func() {
		closeCtx, closeCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer closeCancel()
		_ = conn.Close(closeCtx)
	})

	if _, err := conn.Exec(ctx, `DROP SCHEMA IF EXISTS postgresx_test CASCADE`); err != nil {
		t.Fatalf("reset integration schema: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		_, _ = conn.Exec(cleanupCtx, `DROP SCHEMA IF EXISTS postgresx_test CASCADE`)
		_, _ = conn.Exec(cleanupCtx, `DROP SCHEMA IF EXISTS postgresx_unledgered_test CASCADE`)
		_, _ = conn.Exec(cleanupCtx, `
			DELETE FROM coderoam_meta.schema_migrations
			WHERE scope IN ('postgresx_test_apply', 'postgresx_test_recovery', 'postgresx_test_unledgered')`)
	})

	migrations := []Migration{
		{
			Scope:   "postgresx_test_apply",
			Version: 2,
			Name:    "add_second_row",
			SQL:     `INSERT INTO postgresx_test.items (value) VALUES ('second')`,
		},
		{
			Scope:   "postgresx_test_apply",
			Version: 1,
			Name:    "init",
			SQL: `
				CREATE SCHEMA postgresx_test;
				CREATE TABLE postgresx_test.items (value text PRIMARY KEY);
				INSERT INTO postgresx_test.items (value) VALUES ('first');`,
		},
	}
	if err := ApplyMigrations(ctx, conn, migrations); err != nil {
		t.Fatalf("first ApplyMigrations() error = %v", err)
	}
	if err := ApplyMigrations(ctx, conn, migrations); err != nil {
		t.Fatalf("repeated ApplyMigrations() error = %v", err)
	}

	var itemCount int
	if err := conn.QueryRow(ctx, `SELECT count(*) FROM postgresx_test.items`).Scan(&itemCount); err != nil {
		t.Fatalf("count migrated rows: %v", err)
	}
	if itemCount != 2 {
		t.Fatalf("migrated row count = %d, want 2", itemCount)
	}

	drifted := migrations[1]
	drifted.SQL = `SELECT 1`
	if err := ApplyMigrations(ctx, conn, []Migration{drifted}); !errors.Is(err, ErrMigrationConflict) {
		t.Fatalf("drifted ApplyMigrations() error = %v, want ErrMigrationConflict", err)
	}

	failing := Migration{
		Scope:   "postgresx_test_recovery",
		Version: 1,
		Name:    "init",
		SQL: `
			CREATE TABLE postgresx_test.recovery (value text);
			SELECT postgresx_test_missing_function();`,
	}
	if err := ApplyMigrations(ctx, conn, []Migration{failing}); err == nil {
		t.Fatal("failing ApplyMigrations() error = nil")
	}

	var recoveryTable *string
	if err := conn.QueryRow(ctx, `SELECT to_regclass('postgresx_test.recovery')::text`).Scan(&recoveryTable); err != nil {
		t.Fatalf("inspect rolled-back migration: %v", err)
	}
	if recoveryTable != nil {
		t.Fatalf("failed migration left table %q behind", *recoveryTable)
	}

	recovered := failing
	recovered.SQL = `CREATE TABLE postgresx_test.recovery (value text)`
	if err := ApplyMigrations(ctx, conn, []Migration{recovered}); err != nil {
		t.Fatalf("recovered ApplyMigrations() error = %v", err)
	}

	var recoveryLedgerCount int
	if err := conn.QueryRow(ctx, `
		SELECT count(*)
		FROM coderoam_meta.schema_migrations
		WHERE scope = $1`, recovered.Scope).Scan(&recoveryLedgerCount); err != nil {
		t.Fatalf("count recovery ledger rows: %v", err)
	}
	if recoveryLedgerCount != 1 {
		t.Fatalf("recovery ledger count = %d, want 1", recoveryLedgerCount)
	}

	if _, err := conn.Exec(ctx, `DROP SCHEMA IF EXISTS postgresx_unledgered_test CASCADE`); err != nil {
		t.Fatalf("reset unledgered integration schema: %v", err)
	}
	if _, err := conn.Exec(ctx, `DELETE FROM coderoam_meta.schema_migrations WHERE scope = 'postgresx_test_unledgered'`); err != nil {
		t.Fatalf("reset unledgered migration rows: %v", err)
	}
	if _, err := conn.Exec(ctx, `CREATE SCHEMA postgresx_unledgered_test`); err != nil {
		t.Fatalf("create unledgered integration schema: %v", err)
	}
	unledgered := Migration{
		Scope:   "postgresx_test_unledgered",
		Version: 1,
		Name:    "init",
		SQL: `
			CREATE SCHEMA postgresx_unledgered_test;
			CREATE TABLE postgresx_unledgered_test.items (value text);`,
	}
	err = ApplyMigrations(ctx, conn, []Migration{unledgered})
	var databaseErr *pgconn.PgError
	if !errors.As(err, &databaseErr) || databaseErr.Code != "42P06" {
		t.Fatalf("unledgered ApplyMigrations() error = %v, want duplicate_schema", err)
	}

	var unledgeredTable *string
	if err := conn.QueryRow(ctx, `SELECT to_regclass('postgresx_unledgered_test.items')::text`).Scan(&unledgeredTable); err != nil {
		t.Fatalf("inspect rejected unledgered migration: %v", err)
	}
	if unledgeredTable != nil {
		t.Fatalf("rejected unledgered migration left table %q behind", *unledgeredTable)
	}

	var unledgeredLedgerCount int
	if err := conn.QueryRow(ctx, `
		SELECT count(*) FROM coderoam_meta.schema_migrations
		WHERE scope = $1`, unledgered.Scope).Scan(&unledgeredLedgerCount); err != nil {
		t.Fatalf("count unledgered migration rows: %v", err)
	}
	if unledgeredLedgerCount != 0 {
		t.Fatalf("unledgered migration ledger count = %d, want 0", unledgeredLedgerCount)
	}
}
