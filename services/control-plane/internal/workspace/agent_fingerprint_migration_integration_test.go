package workspace

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hgunduzoglu/coderoam/packages/go/postgresx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestAgentFingerprintMigrationIntegration(t *testing.T) {
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
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		resetAgentFingerprintMigrationState(t, cleanupCtx, conn)
	})

	t.Run("backfills and constrains canonical fingerprints", func(t *testing.T) {
		resetAgentFingerprintMigrationState(t, ctx, conn)
		migrations := agentFingerprintTestMigrations(t)
		if err := postgresx.ApplyMigrations(ctx, conn, migrations[:1]); err != nil {
			t.Fatalf("apply starter workspace migration: %v", err)
		}
		key := bytes.Repeat([]byte{0x42}, 32)
		insertLegacyAgentMigrationRow(t, ctx, conn, strings.Repeat("1", 32), key, "legacy-agent")

		if err := postgresx.ApplyMigrations(ctx, conn, migrations); err != nil {
			t.Fatalf("apply canonical agent fingerprint migration: %v", err)
		}
		if err := postgresx.ApplyMigrations(ctx, conn, migrations); err != nil {
			t.Fatalf("repeat canonical agent fingerprint migration: %v", err)
		}

		var storedFingerprint string
		if err := conn.QueryRow(ctx, `
			SELECT public_key_fingerprint
			FROM workspace_agent_fingerprint_migration_test.agents
			WHERE id = $1`, strings.Repeat("1", 32)).Scan(&storedFingerprint); err != nil {
			t.Fatalf("read backfilled agent fingerprint: %v", err)
		}
		if want := rawAgentFingerprint(key); storedFingerprint != want {
			t.Fatalf("backfilled fingerprint = %q, want %q", storedFingerprint, want)
		}

		assertAgentFingerprintConstraint(
			t, ctx, conn, []string{
				"agents_static_public_key_length", "agents_static_public_key_canonical",
			},
			strings.Repeat("2", 32), bytes.Repeat([]byte{0x43}, 31),
		)
		assertAgentFingerprintConstraint(
			t, ctx, conn, []string{"agents_static_public_key_usable"},
			strings.Repeat("3", 32), make([]byte, 32),
		)
		alias := bytes.Repeat([]byte{0x42}, 32)
		alias[31] |= 0x80
		assertAgentFingerprintConstraint(
			t, ctx, conn, []string{"agents_static_public_key_canonical"},
			strings.Repeat("8", 32), alias,
		)
		lowOrder := make([]byte, 32)
		lowOrder[0] = 1
		assertAgentFingerprintConstraint(
			t, ctx, conn, []string{"agents_static_public_key_usable"},
			strings.Repeat("9", 32), lowOrder,
		)
		_, err := conn.Exec(ctx, `
			UPDATE workspace_agent_fingerprint_migration_test.agents
			SET public_key_fingerprint = $1
			WHERE id = $2`,
			"x25519-sha256:"+strings.Repeat("f", 64), strings.Repeat("1", 32),
		)
		assertAgentFingerprintDatabaseError(t, err, "agents_public_key_fingerprint_matches_key")

		_, err = conn.Exec(ctx, `
			INSERT INTO workspace_agent_fingerprint_migration_test.agents (
				id, user_id, name, static_public_key, public_key_fingerprint, version, created_at
			) VALUES ($1, $2, 'Duplicate agent', $3, $4, '0.1.0', now())`,
			strings.Repeat("4", 32), strings.Repeat("a", 32), key, rawAgentFingerprint(key),
		)
		assertAgentFingerprintDatabaseError(t, err, "agents_public_key_fingerprint_key")
	})

	t.Run("rejects duplicate legacy keys without partial backfill", func(t *testing.T) {
		resetAgentFingerprintMigrationState(t, ctx, conn)
		migrations := agentFingerprintTestMigrations(t)
		if err := postgresx.ApplyMigrations(ctx, conn, migrations[:1]); err != nil {
			t.Fatalf("apply starter workspace migration: %v", err)
		}
		key := bytes.Repeat([]byte{0x51}, 32)
		insertLegacyAgentMigrationRow(t, ctx, conn, strings.Repeat("5", 32), key, "legacy-one")
		insertLegacyAgentMigrationRow(t, ctx, conn, strings.Repeat("6", 32), key, "legacy-two")

		err := postgresx.ApplyMigrations(ctx, conn, migrations)
		assertAgentFingerprintDatabaseError(t, err, "agents_static_public_key_unique")
		assertAgentFingerprintMigrationNotApplied(t, ctx, conn, []string{"legacy-one", "legacy-two"})
	})

	t.Run("rejects invalid legacy keys without partial backfill", func(t *testing.T) {
		highBitAlias := bytes.Repeat([]byte{0x61}, 32)
		highBitAlias[31] |= 0x80
		lowOrder := make([]byte, 32)
		lowOrder[0] = 1
		tests := map[string][]byte{
			"malformed length": bytes.Repeat([]byte{0x61}, 31),
			"high bit alias":   highBitAlias,
			"low order point":  lowOrder,
		}
		for name, key := range tests {
			t.Run(name, func(t *testing.T) {
				resetAgentFingerprintMigrationState(t, ctx, conn)
				migrations := agentFingerprintTestMigrations(t)
				if err := postgresx.ApplyMigrations(ctx, conn, migrations[:1]); err != nil {
					t.Fatalf("apply starter workspace migration: %v", err)
				}
				insertLegacyAgentMigrationRow(
					t, ctx, conn, strings.Repeat("7", 32), key, "legacy-invalid",
				)

				err := postgresx.ApplyMigrations(ctx, conn, migrations)
				assertAgentFingerprintDatabaseError(t, err, "agents_static_public_key_valid")
				assertAgentFingerprintMigrationNotApplied(t, ctx, conn, []string{"legacy-invalid"})
			})
		}
	})
}

func assertAgentFingerprintConstraint(
	t *testing.T,
	ctx context.Context,
	conn *pgx.Conn,
	constraints []string,
	id string,
	key []byte,
) {
	t.Helper()
	_, err := conn.Exec(ctx, `
		INSERT INTO workspace_agent_fingerprint_migration_test.agents (
			id, user_id, name, static_public_key, public_key_fingerprint, version, created_at
		) VALUES ($1, $2, 'Invalid agent', $3, $4, '0.1.0', now())`,
		id, strings.Repeat("a", 32), key, rawAgentFingerprint(key),
	)
	assertAgentFingerprintDatabaseError(t, err, constraints...)
}

func assertAgentFingerprintDatabaseError(t *testing.T, err error, constraints ...string) {
	t.Helper()
	var databaseErr *pgconn.PgError
	if errors.As(err, &databaseErr) {
		for _, constraint := range constraints {
			if databaseErr.ConstraintName == constraint {
				return
			}
		}
	}
	t.Fatalf("database error = %v, want one of constraints %v", err, constraints)
}

func assertAgentFingerprintMigrationNotApplied(
	t *testing.T,
	ctx context.Context,
	conn *pgx.Conn,
	wantFingerprints []string,
) {
	t.Helper()
	var ledgerCount int
	if err := conn.QueryRow(ctx, `
		SELECT count(*) FROM coderoam_meta.schema_migrations
		WHERE scope = 'workspace_agent_fingerprint_migration_test' AND version = 2`,
	).Scan(&ledgerCount); err != nil {
		t.Fatalf("read agent migration ledger: %v", err)
	}
	rows, err := conn.Query(ctx, `
		SELECT public_key_fingerprint
		FROM workspace_agent_fingerprint_migration_test.agents
		ORDER BY public_key_fingerprint`)
	if err != nil {
		t.Fatalf("read legacy agent fingerprints: %v", err)
	}
	defer rows.Close()
	var gotFingerprints []string
	for rows.Next() {
		var fingerprint string
		if err := rows.Scan(&fingerprint); err != nil {
			t.Fatalf("scan legacy agent fingerprint: %v", err)
		}
		gotFingerprints = append(gotFingerprints, fingerprint)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate legacy agent fingerprints: %v", err)
	}
	if ledgerCount != 0 || fmt.Sprint(gotFingerprints) != fmt.Sprint(wantFingerprints) {
		t.Fatalf("failed migration left ledger=%d fingerprints=%v", ledgerCount, gotFingerprints)
	}
}

func insertLegacyAgentMigrationRow(
	t *testing.T,
	ctx context.Context,
	conn *pgx.Conn,
	id string,
	key []byte,
	fingerprint string,
) {
	t.Helper()
	if _, err := conn.Exec(ctx, `
		INSERT INTO workspace_agent_fingerprint_migration_test.agents (
			id, user_id, name, static_public_key, public_key_fingerprint, version, created_at
		) VALUES ($1, $2, 'Legacy agent', $3, $4, '0.1.0', now())`,
		id, strings.Repeat("a", 32), key, fingerprint,
	); err != nil {
		t.Fatalf("insert legacy agent: %v", err)
	}
}

func agentFingerprintTestMigrations(t *testing.T) []postgresx.Migration {
	t.Helper()
	files := []struct {
		version uint64
		name    string
		path    string
	}{
		{version: 1, name: "init", path: "migrations/000001_init.sql"},
		{version: 2, name: "canonical_agent_fingerprint", path: "migrations/000002_canonical_agent_fingerprint.sql"},
	}
	migrations := make([]postgresx.Migration, 0, len(files))
	for _, file := range files {
		sql, err := os.ReadFile(file.path)
		if err != nil {
			t.Fatalf("read workspace migration %d: %v", file.version, err)
		}
		rewritten := strings.Replace(string(sql), "CREATE SCHEMA workspace;", "CREATE SCHEMA workspace_agent_fingerprint_migration_test;", 1)
		rewritten = strings.ReplaceAll(rewritten, "workspace.", "workspace_agent_fingerprint_migration_test.")
		migrations = append(migrations, postgresx.Migration{
			Scope: "workspace_agent_fingerprint_migration_test", Version: file.version,
			Name: file.name, SQL: rewritten,
		})
	}
	return migrations
}

func resetAgentFingerprintMigrationState(t *testing.T, ctx context.Context, conn *pgx.Conn) {
	t.Helper()
	if _, err := conn.Exec(ctx, `
		DROP SCHEMA IF EXISTS workspace_agent_fingerprint_migration_test CASCADE;
		DO $reset$
		BEGIN
		  IF to_regclass('coderoam_meta.schema_migrations') IS NOT NULL THEN
		    DELETE FROM coderoam_meta.schema_migrations
		    WHERE scope = 'workspace_agent_fingerprint_migration_test';
		  END IF;
		END
		$reset$;`); err != nil {
		t.Fatalf("reset agent fingerprint migration state: %v", err)
	}
}

func rawAgentFingerprint(key []byte) string {
	digest := sha256.Sum256(key)
	return fmt.Sprintf("x25519-sha256:%x", digest)
}
