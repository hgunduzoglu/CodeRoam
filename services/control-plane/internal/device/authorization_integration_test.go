package device

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/hgunduzoglu/coderoam/packages/go/postgresx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestAuthorizationIntegration(t *testing.T) {
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
		t.Fatalf("begin device authorization transaction: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if err := tx.Rollback(cleanupCtx); err != nil {
			t.Errorf("rollback device authorization transaction: %v", err)
		}
	})

	owner := newTestActor(t, "0123456789abcdef0123456789abcdef", "owner@example.com")
	foreignActor := newTestActor(t, "2123456789abcdef0123456789abcdef", "foreign@example.com")
	ownerID, _ := owner.UserID()
	checkedAt := time.Date(2026, time.July, 18, 16, 0, 0, 0, time.UTC)
	repository, err := NewRepository(tx, func() time.Time { return checkedAt })
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}

	activeDeviceID := newIntegrationDeviceID(t)
	insertDeviceFixture(t, ctx, tx, activeDeviceID, ownerID.String(), checkedAt.Add(-time.Hour))
	if err := repository.Authorize(ctx, tx, owner, activeDeviceID); err != nil {
		t.Fatalf("Authorize(owner active) error = %v", err)
	}
	assertDeviceRevocation(t, ctx, tx, activeDeviceID, nil, 0)
	if err := repository.Authorize(ctx, tx, foreignActor, activeDeviceID); !errors.Is(err, ErrDeviceAccessDenied) {
		t.Fatalf("Authorize(foreign) error = %v, want ErrDeviceAccessDenied", err)
	}
	if err := repository.Authorize(ctx, tx, owner, newIntegrationDeviceID(t)); !errors.Is(err, ErrDeviceAccessDenied) {
		t.Fatalf("Authorize(missing) error = %v, want ErrDeviceAccessDenied", err)
	}

	futureDeviceID := newIntegrationDeviceID(t)
	insertDeviceFixture(t, ctx, tx, futureDeviceID, ownerID.String(), checkedAt.Add(time.Hour))
	if err := repository.Authorize(ctx, tx, owner, futureDeviceID); !errors.Is(err, ErrDeviceAccessDenied) {
		t.Fatalf("Authorize(future pairing) error = %v, want ErrDeviceAccessDenied", err)
	}

	corruptDeviceID := newIntegrationDeviceID(t)
	insertDeviceFixture(t, ctx, tx, corruptDeviceID, ownerID.String(), checkedAt.Add(-time.Hour))
	if _, err := tx.Exec(ctx, `UPDATE device.devices SET static_public_key = $1 WHERE id = $2`, []byte{0x42}, corruptDeviceID); err != nil {
		t.Fatalf("corrupt stored device public key: %v", err)
	}
	if err := repository.Authorize(ctx, tx, owner, corruptDeviceID); !errors.Is(err, ErrDeviceAccessDenied) {
		t.Fatalf("Authorize(corrupt key) error = %v, want ErrDeviceAccessDenied", err)
	}

	if err := repository.Revoke(ctx, owner, activeDeviceID); err != nil {
		t.Fatalf("Revoke(active device) error = %v", err)
	}
	if err := repository.Authorize(ctx, tx, owner, activeDeviceID); !errors.Is(err, ErrDeviceAccessDenied) {
		t.Fatalf("Authorize(revoked) error = %v, want ErrDeviceAccessDenied", err)
	}
	assertDeviceRevocation(t, ctx, tx, activeDeviceID, &checkedAt, 1)
}

func TestAuthorizationTimeoutIntegration(t *testing.T) {
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

	owner := newTestActor(t, "0123456789abcdef0123456789abcdef", "owner@example.com")
	ownerID, _ := owner.UserID()
	checkedAt := time.Date(2026, time.July, 18, 16, 0, 0, 0, time.UTC)
	deviceID := insertCommittedAuthorizationFixture(t, ctx, pool, ownerID.String(), checkedAt.Add(-time.Hour), 0x44)

	lockingTx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin authorization locking transaction: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if err := lockingTx.Rollback(cleanupCtx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			t.Errorf("rollback authorization locking transaction: %v", err)
		}
	})
	if _, err := lockingTx.Exec(ctx, `LOCK TABLE device.devices IN ACCESS EXCLUSIVE MODE`); err != nil {
		t.Fatalf("lock device table: %v", err)
	}

	repository, err := NewRepository(pool, func() time.Time { return checkedAt })
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}
	repository.operationMax = 100 * time.Millisecond
	authorizationTx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin bounded authorization transaction: %v", err)
	}
	if err := repository.Authorize(context.Background(), authorizationTx, owner, deviceID); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Authorize(locked table) error = %v, want context.DeadlineExceeded", err)
	}
	if err := authorizationTx.Rollback(ctx); err != nil &&
		!errors.Is(err, pgx.ErrTxClosed) && !authorizationTx.Conn().IsClosed() {
		t.Fatalf("rollback bounded authorization transaction: %v", err)
	}
	if err := lockingTx.Rollback(ctx); err != nil {
		t.Fatalf("release device table lock: %v", err)
	}
	assertDeviceRevocation(t, ctx, pool, deviceID, nil, 0)
	retryTx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin authorization retry transaction: %v", err)
	}
	if err := repository.Authorize(ctx, retryTx, owner, deviceID); err != nil {
		t.Fatalf("Authorize(after lock release) error = %v", err)
	}
	if err := retryTx.Commit(ctx); err != nil {
		t.Fatalf("commit authorization retry transaction: %v", err)
	}
	assertDeviceRevocation(t, ctx, pool, deviceID, nil, 0)
}

func TestAuthorizationLockIntegration(t *testing.T) {
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

	owner := newTestActor(t, "0123456789abcdef0123456789abcdef", "owner@example.com")
	ownerID, _ := owner.UserID()
	checkedAt := time.Date(2026, time.July, 18, 16, 0, 0, 0, time.UTC)
	deviceID := insertCommittedAuthorizationFixture(t, ctx, pool, ownerID.String(), checkedAt.Add(-time.Hour), 0x45)
	repository, err := NewRepository(pool, func() time.Time { return checkedAt })
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}
	repository.operationMax = 100 * time.Millisecond

	authorizationTx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin active authorization transaction: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if err := authorizationTx.Rollback(cleanupCtx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			t.Errorf("rollback active authorization transaction: %v", err)
		}
	})
	if err := repository.Authorize(ctx, authorizationTx, owner, deviceID); err != nil {
		t.Fatalf("Authorize(active transaction) error = %v", err)
	}
	if err := repository.Revoke(context.Background(), owner, deviceID); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Revoke(while authorized transaction open) error = %v, want context.DeadlineExceeded", err)
	}
	assertDeviceRevocation(t, ctx, pool, deviceID, nil, 0)
	if err := authorizationTx.Commit(ctx); err != nil {
		t.Fatalf("commit active authorization transaction: %v", err)
	}
	if err := repository.Revoke(ctx, owner, deviceID); err != nil {
		t.Fatalf("Revoke(after authorization commit) error = %v", err)
	}
	assertDeviceRevocation(t, ctx, pool, deviceID, &checkedAt, 1)
}

func insertCommittedAuthorizationFixture(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	ownerID string,
	pairedAt time.Time,
	keyByte byte,
) string {
	t.Helper()
	deviceID := newIntegrationDeviceID(t)
	if _, err := pool.Exec(ctx, `
		INSERT INTO device.devices (
			id, user_id, name, platform, static_public_key, public_key_fingerprint, paired_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		deviceID,
		ownerID,
		"Authorization integration device",
		"android",
		bytes.Repeat([]byte{keyByte}, 32),
		"fixture:"+deviceID,
		pairedAt,
	); err != nil {
		t.Fatalf("insert authorization fixture: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if _, err := pool.Exec(cleanupCtx, `DELETE FROM outbox.events WHERE aggregate_id = $1`, deviceID); err != nil {
			t.Errorf("delete authorization outbox fixture: %v", err)
		}
		if _, err := pool.Exec(cleanupCtx, `DELETE FROM device.devices WHERE id = $1`, deviceID); err != nil {
			t.Errorf("delete authorization device fixture: %v", err)
		}
	})
	return deviceID
}
