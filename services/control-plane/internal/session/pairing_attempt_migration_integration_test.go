package session

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hgunduzoglu/coderoam/packages/go/postgresx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestPairingAttemptMigrationIntegration(t *testing.T) {
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
	resetPairingAttemptMigrationState(t, ctx, conn)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		resetPairingAttemptMigrationState(t, cleanupCtx, conn)
	})

	migrations := pairingAttemptTestMigrations(t)
	if err := postgresx.ApplyMigrations(ctx, conn, migrations[:1]); err != nil {
		t.Fatalf("apply starter session migration: %v", err)
	}
	if _, err := conn.Exec(ctx, `
		INSERT INTO session_pairing_migration_test.pairing_attempts (
			id, agent_fingerprint, expires_at, attempt_count, created_at
		) VALUES ('legacy-attempt', 'legacy-fingerprint', $1, 0, $2)`,
		time.Date(2026, time.July, 31, 13, 5, 0, 0, time.UTC),
		time.Date(2026, time.July, 31, 13, 0, 0, 0, time.UTC),
	); err != nil {
		t.Fatalf("insert starter pairing attempt: %v", err)
	}

	if err := postgresx.ApplyMigrations(ctx, conn, migrations); err != nil {
		t.Fatalf("apply M3 pairing-attempt migration: %v", err)
	}
	if err := postgresx.ApplyMigrations(ctx, conn, migrations); err != nil {
		t.Fatalf("repeat M3 pairing-attempt migration: %v", err)
	}

	var legacyCount int
	if err := conn.QueryRow(ctx, `
		SELECT count(*) FROM session_pairing_migration_test.pairing_attempts
		WHERE id = 'legacy-attempt'`,
	).Scan(&legacyCount); err != nil {
		t.Fatalf("count legacy pairing attempts: %v", err)
	}
	if legacyCount != 0 {
		t.Fatalf("legacy pairing-attempt count = %d, want 0", legacyCount)
	}

	createdAt := time.Date(2026, time.July, 31, 13, 0, 0, 0, time.UTC)
	valid := newPairingAttemptMigrationRow("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", createdAt)
	if err := insertPairingAttemptMigrationRow(ctx, conn, valid); err != nil {
		t.Fatalf("insert valid M3 pairing attempt: %v", err)
	}
	assertPairingAttemptMigrationUpdateRejected(
		t, ctx, conn, "pairing_attempts_claim_shape",
		`UPDATE session_pairing_migration_test.pairing_attempts
		 SET state = 'claimed',
		     claimed_user_id = '11111111111111111111111111111111',
		     device_id = '22222222222222222222222222222222',
		     device_display_name = 'Owner phone',
		     device_platform = NULL,
		     device_static_public_key = $2,
		     device_key_fingerprint = $3,
		     claimed_at = $4,
		     updated_at = $4
		 WHERE id = $1`,
		valid.id, bytes.Repeat([]byte{0x31}, 32),
		"x25519-sha256:"+strings.Repeat("2", 64), createdAt.Add(time.Minute),
	)
	assertPairingAttemptMigrationAllowsValidLifecycle(t, ctx, conn, valid.id, createdAt)
	assertPairingAttemptMigrationUpdateRejected(
		t, ctx, conn, "pairing_attempts_channel_binding_match",
		`UPDATE session_pairing_migration_test.pairing_attempts
		 SET agent_channel_binding = $2 WHERE id = $1`,
		valid.id, bytes.Repeat([]byte{0x52}, 32),
	)
	assertPairingAttemptMigrationUpdateRejected(
		t, ctx, conn, "pairing_attempts_transition_timestamps",
		`UPDATE session_pairing_migration_test.pairing_attempts
		 SET consumed_at = $2 WHERE id = $1`,
		valid.id, createdAt.Add(90*time.Second),
	)

	tests := []struct {
		name       string
		constraint string
		mutate     func(*pairingAttemptMigrationRow)
	}{
		{
			name: "invalid agent id", constraint: "pairing_attempts_agent_id_shape",
			mutate: func(row *pairingAttemptMigrationRow) { row.agentID = "invalid" },
		},
		{
			name: "short agent key", constraint: "pairing_attempts_agent_key_length",
			mutate: func(row *pairingAttemptMigrationRow) { row.agentPublicKey = bytes.Repeat([]byte{1}, 31) },
		},
		{
			name: "noncanonical fingerprint", constraint: "pairing_attempts_agent_fingerprint_shape",
			mutate: func(row *pairingAttemptMigrationRow) {
				row.agentFingerprint = "x25519-sha256:" + strings.Repeat("A", 64)
			},
		},
		{
			name: "excess failures", constraint: "pairing_attempts_failure_count_range",
			mutate: func(row *pairingAttemptMigrationRow) { row.failedAttemptCount = 9 },
		},
		{
			name: "excess lifetime", constraint: "pairing_attempts_lifetime",
			mutate: func(row *pairingAttemptMigrationRow) { row.expiresAt = row.createdAt.Add(5*time.Minute + time.Second) },
		},
		{
			name: "unknown state", constraint: "pairing_attempts_state_value",
			mutate: func(row *pairingAttemptMigrationRow) { row.state = "expired" },
		},
		{
			name: "consumed without claim", constraint: "pairing_attempts_state_shape",
			mutate: func(row *pairingAttemptMigrationRow) { row.state = "consumed" },
		},
	}
	fixtureIDs := []string{
		strings.Repeat("b", 32), strings.Repeat("c", 32), strings.Repeat("d", 32),
		strings.Repeat("e", 32), strings.Repeat("f", 32), strings.Repeat("0", 32),
		strings.Repeat("9", 32),
	}
	for index, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			row := newPairingAttemptMigrationRow(fixtureIDs[index], createdAt)
			test.mutate(&row)
			err := insertPairingAttemptMigrationRow(ctx, conn, row)
			var databaseErr *pgconn.PgError
			if !errors.As(err, &databaseErr) || databaseErr.ConstraintName != test.constraint {
				t.Fatalf("invalid pairing-attempt error = %v, want constraint %q", err, test.constraint)
			}
		})
	}

	assertPairingAttemptMigrationStoresNoSecret(t, ctx, conn)
	assertPairingAttemptMigrationRollsBack(t, ctx, conn)
}

func assertPairingAttemptMigrationUpdateRejected(
	t *testing.T,
	ctx context.Context,
	conn *pgx.Conn,
	constraint string,
	query string,
	arguments ...any,
) {
	t.Helper()
	_, err := conn.Exec(ctx, query, arguments...)
	var databaseErr *pgconn.PgError
	if !errors.As(err, &databaseErr) || databaseErr.ConstraintName != constraint {
		t.Fatalf("invalid pairing-attempt update error = %v, want constraint %q", err, constraint)
	}
}

func assertPairingAttemptMigrationAllowsValidLifecycle(
	t *testing.T,
	ctx context.Context,
	conn *pgx.Conn,
	id string,
	createdAt time.Time,
) {
	t.Helper()
	claimedAt := createdAt.Add(time.Minute)
	if _, err := conn.Exec(ctx, `
		UPDATE session_pairing_migration_test.pairing_attempts
		SET state = 'claimed',
		    claimed_user_id = '11111111111111111111111111111111',
		    device_id = '22222222222222222222222222222222',
		    device_display_name = 'Owner phone',
		    device_platform = 'ios',
		    device_static_public_key = $2,
		    device_key_fingerprint = $3,
		    claimed_at = $4,
		    updated_at = $4
		WHERE id = $1`,
		id, bytes.Repeat([]byte{0x31}, 32),
		"x25519-sha256:"+strings.Repeat("2", 64), claimedAt,
	); err != nil {
		t.Fatalf("persist valid pairing claim: %v", err)
	}

	channelBinding := bytes.Repeat([]byte{0x51}, 32)
	mobileConfirmedAt := createdAt.Add(2 * time.Minute)
	if _, err := conn.Exec(ctx, `
		UPDATE session_pairing_migration_test.pairing_attempts
		SET state = 'confirming', mobile_channel_binding = $2,
		    mobile_confirmed_at = $3, updated_at = $3
		WHERE id = $1`, id, channelBinding, mobileConfirmedAt); err != nil {
		t.Fatalf("persist valid mobile confirmation: %v", err)
	}

	agentConfirmedAt := createdAt.Add(3 * time.Minute)
	if _, err := conn.Exec(ctx, `
		UPDATE session_pairing_migration_test.pairing_attempts
		SET agent_channel_binding = $2, agent_confirmed_at = $3, updated_at = $3
		WHERE id = $1`, id, channelBinding, agentConfirmedAt); err != nil {
		t.Fatalf("persist valid agent confirmation: %v", err)
	}

	consumedAt := createdAt.Add(4 * time.Minute)
	if _, err := conn.Exec(ctx, `
		UPDATE session_pairing_migration_test.pairing_attempts
		SET state = 'consumed', consumed_at = $2, updated_at = $2
		WHERE id = $1`, id, consumedAt); err != nil {
		t.Fatalf("persist valid consumed attempt: %v", err)
	}
}

type pairingAttemptMigrationRow struct {
	id                 string
	agentID            string
	agentPublicKey     []byte
	agentFingerprint   string
	agentDisplayName   string
	agentVersion       string
	protocolVersion    int
	relayRegion        string
	bootstrapHash      []byte
	expiresAt          time.Time
	failedAttemptCount int
	state              string
	createdAt          time.Time
	updatedAt          time.Time
}

func newPairingAttemptMigrationRow(id string, createdAt time.Time) pairingAttemptMigrationRow {
	return pairingAttemptMigrationRow{
		id: id, agentID: strings.Repeat("8", 32), agentPublicKey: bytes.Repeat([]byte{0x42}, 32),
		agentFingerprint: "x25519-sha256:" + strings.Repeat("1", 64),
		agentDisplayName: "M3 agent", agentVersion: "0.1.0", protocolVersion: 1,
		relayRegion: "eu-test-1", bootstrapHash: bytes.Repeat([]byte{0x24}, 32),
		expiresAt: createdAt.Add(5 * time.Minute), failedAttemptCount: 0,
		state: "open", createdAt: createdAt, updatedAt: createdAt,
	}
}

func insertPairingAttemptMigrationRow(
	ctx context.Context,
	conn *pgx.Conn,
	row pairingAttemptMigrationRow,
) error {
	_, err := conn.Exec(ctx, `
		INSERT INTO session_pairing_migration_test.pairing_attempts (
			id, agent_id, agent_static_public_key, agent_key_fingerprint, agent_display_name,
			agent_version, protocol_version, relay_region, bootstrap_credential_hash,
			expires_at, failed_attempt_count, state, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
		row.id, row.agentID, row.agentPublicKey, row.agentFingerprint, row.agentDisplayName,
		row.agentVersion, row.protocolVersion, row.relayRegion, row.bootstrapHash,
		row.expiresAt, row.failedAttemptCount, row.state, row.createdAt, row.updatedAt,
	)
	return err
}

func assertPairingAttemptMigrationStoresNoSecret(t *testing.T, ctx context.Context, conn *pgx.Conn) {
	t.Helper()
	rows, err := conn.Query(ctx, `
		SELECT column_name
		FROM information_schema.columns
		WHERE table_schema = 'session_pairing_migration_test'
		  AND table_name = 'pairing_attempts'
		  AND column_name IN ('pairing_secret', 'bootstrap_credential')`)
	if err != nil {
		t.Fatalf("inspect pairing-attempt secret columns: %v", err)
	}
	defer rows.Close()
	if rows.Next() {
		var column string
		if err := rows.Scan(&column); err != nil {
			t.Fatalf("scan forbidden pairing-attempt column: %v", err)
		}
		t.Fatalf("pairing-attempt migration persisted forbidden secret column %q", column)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate pairing-attempt columns: %v", err)
	}
}

func assertPairingAttemptMigrationRollsBack(t *testing.T, ctx context.Context, conn *pgx.Conn) {
	t.Helper()
	failing := postgresx.Migration{
		Scope: "session_pairing_migration_test", Version: 4, Name: "rollback_probe",
		SQL: `
			ALTER TABLE session_pairing_migration_test.pairing_attempts
			  ADD COLUMN rollback_probe text;
			SELECT session_pairing_migration_test.missing_pairing_migration_function();`,
	}
	if err := postgresx.ApplyMigrations(ctx, conn, []postgresx.Migration{failing}); err == nil {
		t.Fatal("failing pairing-attempt migration error = nil")
	}

	var columnCount, ledgerCount int
	if err := conn.QueryRow(ctx, `
		SELECT count(*)
		FROM information_schema.columns
		WHERE table_schema = 'session_pairing_migration_test'
		  AND table_name = 'pairing_attempts'
		  AND column_name = 'rollback_probe'`).Scan(&columnCount); err != nil {
		t.Fatalf("inspect rolled-back pairing-attempt column: %v", err)
	}
	if err := conn.QueryRow(ctx, `
		SELECT count(*)
		FROM coderoam_meta.schema_migrations
		WHERE scope = 'session_pairing_migration_test' AND version = 4`).Scan(&ledgerCount); err != nil {
		t.Fatalf("inspect rolled-back pairing-attempt ledger: %v", err)
	}
	if columnCount != 0 || ledgerCount != 0 {
		t.Fatalf("failed migration left column=%d ledger=%d, want zero", columnCount, ledgerCount)
	}

	recovered := failing
	recovered.SQL = `ALTER TABLE session_pairing_migration_test.pairing_attempts ADD COLUMN rollback_probe text`
	if err := postgresx.ApplyMigrations(ctx, conn, []postgresx.Migration{recovered}); err != nil {
		t.Fatalf("recover pairing-attempt migration: %v", err)
	}
}

func resetPairingAttemptMigrationState(t *testing.T, ctx context.Context, conn *pgx.Conn) {
	t.Helper()
	if _, err := conn.Exec(ctx, `
		DROP SCHEMA IF EXISTS session_pairing_migration_test CASCADE;
		DO $reset$
		BEGIN
		  IF to_regclass('coderoam_meta.schema_migrations') IS NOT NULL THEN
		    DELETE FROM coderoam_meta.schema_migrations
		    WHERE scope = 'session_pairing_migration_test';
		  END IF;
		END
		$reset$;`); err != nil {
		t.Fatalf("reset pairing-attempt migration state: %v", err)
	}
}

func pairingAttemptTestMigrations(t *testing.T) []postgresx.Migration {
	t.Helper()
	migrations := readSessionIntegrationMigrations(t)
	for index := range migrations {
		migrations[index].Scope = "session_pairing_migration_test"
		migrations[index].SQL = strings.Replace(
			migrations[index].SQL,
			"CREATE SCHEMA session;",
			"CREATE SCHEMA session_pairing_migration_test;",
			1,
		)
		migrations[index].SQL = strings.ReplaceAll(
			migrations[index].SQL,
			"session.",
			"session_pairing_migration_test.",
		)
	}
	return migrations
}
