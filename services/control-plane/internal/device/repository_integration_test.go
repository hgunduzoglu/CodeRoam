package device

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/hgunduzoglu/coderoam/packages/go/ids"
	"github.com/hgunduzoglu/coderoam/packages/go/postgresx"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/outbox"
	"github.com/jackc/pgx/v5"
)

func TestRepositoryIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_TEST_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_TEST_DSN is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)
	pool, err := postgresx.OpenPool(ctx, dsn)
	if err != nil {
		t.Fatalf("OpenPool() error = %v", err)
	}
	t.Cleanup(pool.Close)
	applyDeviceIntegrationMigrations(t, ctx, pool)

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin device repository transaction: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if err := tx.Rollback(cleanupCtx); err != nil {
			t.Errorf("rollback device repository transaction: %v", err)
		}
	})

	owner := newTestActor(t, "0123456789abcdef0123456789abcdef", "owner@example.com")
	foreignActor := newTestActor(t, "2123456789abcdef0123456789abcdef", "foreign@example.com")
	ownerID, _ := owner.UserID()
	revokedAt := time.Date(2026, time.July, 18, 15, 0, 0, 0, time.UTC)
	pairedAt := revokedAt.Add(-time.Hour)
	repository, err := NewRepository(tx, func() time.Time { return revokedAt })
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}

	ownedDeviceID := newIntegrationDeviceID(t)
	insertDeviceFixture(t, ctx, tx, ownedDeviceID, ownerID.String(), pairedAt)
	if err := repository.Revoke(ctx, owner, ownedDeviceID); err != nil {
		t.Fatalf("Revoke(owner) error = %v", err)
	}
	assertDeviceRevocation(t, ctx, tx, ownedDeviceID, &revokedAt, 1)
	var eventType, aggregateType, payload string
	var availableAt time.Time
	if err := tx.QueryRow(ctx, `
		SELECT event_type, aggregate_type, payload::text, available_at
		FROM outbox.events
		WHERE aggregate_id = $1`, ownedDeviceID).Scan(&eventType, &aggregateType, &payload, &availableAt); err != nil {
		t.Fatalf("read device revocation event: %v", err)
	}
	if eventType != "device.revoked.v1" || aggregateType != "device" || payload != "{}" || !availableAt.Equal(revokedAt) {
		t.Fatal("device revocation did not preserve the metadata-only outbox contract")
	}
	if err := repository.Revoke(ctx, owner, ownedDeviceID); err != nil {
		t.Fatalf("Revoke(owner repeated) error = %v", err)
	}
	assertDeviceRevocation(t, ctx, tx, ownedDeviceID, &revokedAt, 1)

	foreignDeviceID := newIntegrationDeviceID(t)
	insertDeviceFixture(t, ctx, tx, foreignDeviceID, ownerID.String(), pairedAt)
	if err := repository.Revoke(ctx, foreignActor, foreignDeviceID); !errors.Is(err, ErrDeviceAccessDenied) {
		t.Fatalf("Revoke(foreign) error = %v, want ErrDeviceAccessDenied", err)
	}
	assertDeviceRevocation(t, ctx, tx, foreignDeviceID, nil, 0)
	if err := repository.Revoke(ctx, owner, newIntegrationDeviceID(t)); !errors.Is(err, ErrDeviceAccessDenied) {
		t.Fatalf("Revoke(missing) error = %v, want ErrDeviceAccessDenied", err)
	}

	corruptDeviceID := newIntegrationDeviceID(t)
	insertDeviceFixture(t, ctx, tx, corruptDeviceID, "not-an-owner-id", pairedAt)
	if err := repository.Revoke(ctx, owner, corruptDeviceID); !errors.Is(err, ErrDeviceAccessDenied) {
		t.Fatalf("Revoke(corrupt owner) error = %v, want ErrDeviceAccessDenied", err)
	}
	assertDeviceRevocation(t, ctx, tx, corruptDeviceID, nil, 0)

	futureDeviceID := newIntegrationDeviceID(t)
	insertDeviceFixture(t, ctx, tx, futureDeviceID, ownerID.String(), revokedAt.Add(time.Hour))
	if err := repository.Revoke(ctx, owner, futureDeviceID); !errors.Is(err, ErrInvalidDevice) {
		t.Fatalf("Revoke(future pairing) error = %v, want ErrInvalidDevice", err)
	}
	assertDeviceRevocation(t, ctx, tx, futureDeviceID, nil, 0)

	recoveryDeviceID := newIntegrationDeviceID(t)
	insertDeviceFixture(t, ctx, tx, recoveryDeviceID, ownerID.String(), pairedAt)
	repository.enqueue = func(context.Context, pgx.Tx, outbox.Event) error {
		return errors.New("forced outbox failure")
	}
	if err := repository.Revoke(ctx, owner, recoveryDeviceID); !errors.Is(err, ErrDevicePersistenceUnavailable) {
		t.Fatalf("Revoke(outbox failure) error = %v, want ErrDevicePersistenceUnavailable", err)
	}
	assertDeviceRevocation(t, ctx, tx, recoveryDeviceID, nil, 0)
	repository.enqueue = outbox.Enqueue
	if err := repository.Revoke(ctx, owner, recoveryDeviceID); err != nil {
		t.Fatalf("Revoke(recovery) error = %v", err)
	}
	assertDeviceRevocation(t, ctx, tx, recoveryDeviceID, &revokedAt, 1)

	lockedDeviceID := newIntegrationDeviceID(t)
	if _, err := pool.Exec(ctx, `
		INSERT INTO device.devices (
			id, user_id, name, platform, static_public_key, public_key_fingerprint, paired_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		lockedDeviceID,
		ownerID.String(),
		"Locked integration device",
		"ios",
		bytes.Repeat([]byte{0x43}, 32),
		"fixture:"+lockedDeviceID,
		pairedAt,
	); err != nil {
		t.Fatalf("insert locked device fixture: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if _, err := pool.Exec(cleanupCtx, `DELETE FROM outbox.events WHERE aggregate_id = $1`, lockedDeviceID); err != nil {
			t.Errorf("delete locked-device outbox fixture: %v", err)
		}
		if _, err := pool.Exec(cleanupCtx, `DELETE FROM device.devices WHERE id = $1`, lockedDeviceID); err != nil {
			t.Errorf("delete locked device fixture: %v", err)
		}
	})
	lockingTx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin device locking transaction: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if err := lockingTx.Rollback(cleanupCtx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			t.Errorf("rollback device locking transaction: %v", err)
		}
	})
	var selectedDeviceID string
	if err := lockingTx.QueryRow(ctx, `
		SELECT id FROM device.devices WHERE id = $1 FOR UPDATE`, lockedDeviceID).Scan(&selectedDeviceID); err != nil {
		t.Fatalf("lock device fixture: %v", err)
	}
	boundedRepository, err := NewRepository(pool, func() time.Time { return revokedAt })
	if err != nil {
		t.Fatalf("NewRepository(bounded) error = %v", err)
	}
	boundedRepository.operationMax = 100 * time.Millisecond
	if err := boundedRepository.Revoke(context.Background(), foreignActor, lockedDeviceID); !errors.Is(err, ErrDeviceAccessDenied) {
		t.Fatalf("Revoke(foreign locked row) error = %v, want ErrDeviceAccessDenied", err)
	}
	if err := boundedRepository.Revoke(context.Background(), owner, lockedDeviceID); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Revoke(locked row) error = %v, want context.DeadlineExceeded", err)
	}
	assertDeviceRevocation(t, ctx, pool, lockedDeviceID, nil, 0)
	if err := lockingTx.Rollback(ctx); err != nil {
		t.Fatalf("release device row lock: %v", err)
	}
	if err := boundedRepository.Revoke(ctx, owner, lockedDeviceID); err != nil {
		t.Fatalf("Revoke(after lock release) error = %v", err)
	}
	assertDeviceRevocation(t, ctx, pool, lockedDeviceID, &revokedAt, 1)
}

func applyDeviceIntegrationMigrations(t *testing.T, ctx context.Context, database postgresx.TransactionStarter) {
	t.Helper()
	for _, migration := range []struct {
		scope string
		path  string
	}{
		{scope: "device", path: "migrations/000001_init.sql"},
		{scope: "outbox", path: "../outbox/migrations/000001_init.sql"},
	} {
		sql, err := os.ReadFile(migration.path)
		if err != nil {
			t.Fatalf("read %s migration: %v", migration.scope, err)
		}
		if err := postgresx.ApplyMigrations(ctx, database, []postgresx.Migration{{
			Scope: migration.scope, Version: 1, Name: "init", SQL: string(sql),
		}}); err != nil {
			t.Fatalf("apply %s migration: %v", migration.scope, err)
		}
	}
}

func insertDeviceFixture(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	deviceID string,
	ownerID string,
	pairedAt time.Time,
) {
	t.Helper()
	if _, err := tx.Exec(ctx, `
		INSERT INTO device.devices (
			id, user_id, name, platform, static_public_key, public_key_fingerprint, paired_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		deviceID,
		ownerID,
		"Integration device",
		"ios",
		bytes.Repeat([]byte{0x42}, 32),
		"fixture:"+deviceID,
		pairedAt,
	); err != nil {
		t.Fatalf("insert device fixture: %v", err)
	}
}

type deviceStateReader interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func assertDeviceRevocation(
	t *testing.T,
	ctx context.Context,
	reader deviceStateReader,
	deviceID string,
	wantRevokedAt *time.Time,
	wantEventCount int,
) {
	t.Helper()
	var storedRevokedAt *time.Time
	if err := reader.QueryRow(ctx, `SELECT revoked_at FROM device.devices WHERE id = $1`, deviceID).Scan(&storedRevokedAt); err != nil {
		t.Fatalf("read device revocation: %v", err)
	}
	if wantRevokedAt == nil && storedRevokedAt != nil {
		t.Fatalf("revoked_at = %v, want nil", storedRevokedAt)
	}
	if wantRevokedAt != nil && (storedRevokedAt == nil || !storedRevokedAt.Equal(*wantRevokedAt)) {
		t.Fatalf("revoked_at = %v, want %v", storedRevokedAt, wantRevokedAt)
	}
	var eventCount int
	if err := reader.QueryRow(ctx, `
		SELECT count(*) FROM outbox.events
		WHERE aggregate_type = 'device' AND aggregate_id = $1`, deviceID).Scan(&eventCount); err != nil {
		t.Fatalf("count device revocation events: %v", err)
	}
	if eventCount != wantEventCount {
		t.Fatalf("device revocation event count = %d, want %d", eventCount, wantEventCount)
	}
}

func newIntegrationDeviceID(t *testing.T) string {
	t.Helper()
	id, err := ids.New()
	if err != nil {
		t.Fatalf("ids.New() error = %v", err)
	}
	return id.String()
}
