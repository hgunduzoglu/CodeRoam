package main

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"testing"
	"testing/fstest"
	"time"

	"github.com/hgunduzoglu/coderoam/packages/go/postgresx"
	"github.com/jackc/pgx/v5"
)

func TestLoadMigrations(t *testing.T) {
	root := fstest.MapFS{
		"auth/000002_add_index.sql": &fstest.MapFile{Data: []byte("CREATE INDEX users_email_idx ON auth.users (email)")},
		"auth/000001_init.sql":      &fstest.MapFile{Data: []byte("CREATE SCHEMA auth")},
		"auth/README.md":            &fstest.MapFile{Data: []byte("ignored")},
		"device/000001_init.sql":    &fstest.MapFile{Data: []byte("CREATE SCHEMA device")},
	}

	migrations, err := loadMigrations(root, []string{"auth=auth", "device=device"})
	if err != nil {
		t.Fatalf("loadMigrations() error = %v", err)
	}
	if len(migrations) != 3 {
		t.Fatalf("loadMigrations() count = %d, want 3", len(migrations))
	}
	want := []struct {
		scope   string
		version uint64
		name    string
	}{
		{scope: "auth", version: 1, name: "init"},
		{scope: "auth", version: 2, name: "add_index"},
		{scope: "device", version: 1, name: "init"},
	}
	for index, expected := range want {
		migration := migrations[index]
		if migration.Scope != expected.scope || migration.Version != expected.version || migration.Name != expected.name {
			t.Errorf("migration %d = %q/%d/%q, want %q/%d/%q", index, migration.Scope, migration.Version, migration.Name, expected.scope, expected.version, expected.name)
		}
	}
}

func TestLoadMigrationsRejectsInvalidCatalogs(t *testing.T) {
	tests := map[string]struct {
		root  fs.FS
		specs []string
	}{
		"no scopes": {},
		"malformed scope directory": {
			root:  fstest.MapFS{},
			specs: []string{"auth"},
		},
		"duplicate scope": {
			root: fstest.MapFS{
				"one/000001_init.sql": &fstest.MapFile{Data: []byte("SELECT 1")},
				"two/000001_init.sql": &fstest.MapFile{Data: []byte("SELECT 1")},
			},
			specs: []string{"auth=one", "auth=two"},
		},
		"invalid filename": {
			root: fstest.MapFS{
				"auth/1_init.sql": &fstest.MapFile{Data: []byte("SELECT 1")},
			},
			specs: []string{"auth=auth"},
		},
		"zero version": {
			root: fstest.MapFS{
				"auth/000000_init.sql": &fstest.MapFile{Data: []byte("SELECT 1")},
			},
			specs: []string{"auth=auth"},
		},
		"empty migration": {
			root: fstest.MapFS{
				"auth/000001_init.sql": &fstest.MapFile{Data: []byte(" \n")},
			},
			specs: []string{"auth=auth"},
		},
		"symlink migration": {
			root: fstest.MapFS{
				"auth/000001_init.sql": &fstest.MapFile{Mode: fs.ModeSymlink},
			},
			specs: []string{"auth=auth"},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := loadMigrations(test.root, test.specs)
			if !errors.Is(err, postgresx.ErrInvalidMigration) {
				t.Fatalf("loadMigrations() error = %v, want ErrInvalidMigration", err)
			}
		})
	}
}

func TestRunRequiresDSN(t *testing.T) {
	err := run(context.Background(), "", fstest.MapFS{}, []string{"auth=auth"})
	if err == nil || err.Error() != "POSTGRES_DSN is required" {
		t.Fatalf("run() error = %v, want POSTGRES_DSN requirement", err)
	}
}

func TestRunIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_TEST_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_TEST_DSN is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	admin, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect to test PostgreSQL: %v", err)
	}
	t.Cleanup(func() {
		closeCtx, closeCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer closeCancel()
		_ = admin.Close(closeCtx)
	})

	cleanup := func(cleanupCtx context.Context) error {
		if _, err := admin.Exec(cleanupCtx, `DROP SCHEMA IF EXISTS migration_runner_test CASCADE`); err != nil {
			return err
		}
		_, err := admin.Exec(cleanupCtx, `
			DO $$
			BEGIN
				IF to_regclass('coderoam_meta.schema_migrations') IS NOT NULL THEN
					DELETE FROM coderoam_meta.schema_migrations WHERE scope = 'migration_runner_test';
				END IF;
			END
			$$`)
		return err
	}
	if err := cleanup(ctx); err != nil {
		t.Fatalf("reset migration runner integration state: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if err := cleanup(cleanupCtx); err != nil {
			t.Errorf("clean migration runner integration state: %v", err)
		}
	})

	root := fstest.MapFS{
		"migrations/000001_init.sql": &fstest.MapFile{Data: []byte(`
			CREATE SCHEMA migration_runner_test;
			CREATE TABLE migration_runner_test.items (value text PRIMARY KEY);
			INSERT INTO migration_runner_test.items (value) VALUES ('once');`)},
	}
	specs := []string{"migration_runner_test=migrations"}
	if err := run(ctx, dsn, root, specs); err != nil {
		t.Fatalf("first run() error = %v", err)
	}
	if err := run(ctx, dsn, root, specs); err != nil {
		t.Fatalf("repeated run() error = %v", err)
	}

	var itemCount int
	if err := admin.QueryRow(ctx, `SELECT count(*) FROM migration_runner_test.items`).Scan(&itemCount); err != nil {
		t.Fatalf("count migrated rows: %v", err)
	}
	if itemCount != 1 {
		t.Fatalf("migrated row count = %d, want 1", itemCount)
	}

	var ledgerCount int
	if err := admin.QueryRow(ctx, `
		SELECT count(*) FROM coderoam_meta.schema_migrations
		WHERE scope = 'migration_runner_test' AND version = 1`).Scan(&ledgerCount); err != nil {
		t.Fatalf("count runner ledger rows: %v", err)
	}
	if ledgerCount != 1 {
		t.Fatalf("runner ledger count = %d, want 1", ledgerCount)
	}
}
