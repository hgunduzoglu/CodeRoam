package device

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/hgunduzoglu/coderoam/packages/go/cryptox"
	"github.com/hgunduzoglu/coderoam/packages/go/postgresx"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/auth"
	"github.com/jackc/pgx/v5"
)

func TestRepositoryRegisterPairedIntegration(t *testing.T) {
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
	firstTx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin device registration transaction: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if err := firstTx.Rollback(cleanupCtx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			t.Errorf("rollback first device registration transaction: %v", err)
		}
	})

	ownerID := parseDeviceRegistrationOwner(t, "0123456789abcdef0123456789abcdef")
	foreignOwnerID := parseDeviceRegistrationOwner(t, "2123456789abcdef0123456789abcdef")
	pairedAt := time.Date(2026, time.August, 3, 11, 0, 0, 123456789, time.UTC)
	repository, err := NewRepository(pool, func() time.Time { return pairedAt.Add(time.Hour) })
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}
	deviceID := newIntegrationDeviceID(t)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if _, err := pool.Exec(cleanupCtx, `DELETE FROM device.devices WHERE id = $1`, deviceID); err != nil {
			t.Errorf("delete committed paired device: %v", err)
		}
	})
	keyBytes := canonicalDeviceIntegrationPublicKey(deviceID)
	publicKey, err := cryptox.ParseX25519PublicKey(keyBytes[:])
	if err != nil {
		t.Fatalf("ParseX25519PublicKey() error = %v", err)
	}

	register := func(tx pgx.Tx, owner auth.UserID, id, name string, key cryptox.X25519PublicKey) error {
		return repository.RegisterPaired(
			ctx, tx, owner, id, name, PlatformIOS, key, pairedAt,
		)
	}
	if err := register(firstTx, ownerID, deviceID, "Husam's iPhone", publicKey); err != nil {
		t.Fatalf("RegisterPaired(new) error = %v", err)
	}
	if err := firstTx.Commit(ctx); err != nil {
		t.Fatalf("commit first device registration: %v", err)
	}

	retryTx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin device retry transaction: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if err := retryTx.Rollback(cleanupCtx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			t.Errorf("rollback device retry transaction: %v", err)
		}
	})
	if err := register(retryTx, ownerID, deviceID, "Husam's iPhone", publicKey); err != nil {
		t.Fatalf("RegisterPaired(committed exact retry) error = %v", err)
	}
	var deviceCount int
	if err := retryTx.QueryRow(ctx, `SELECT count(*) FROM device.devices WHERE id = $1`, deviceID).Scan(&deviceCount); err != nil {
		t.Fatalf("count paired devices: %v", err)
	}
	if deviceCount != 1 {
		t.Fatalf("paired device count = %d, want 1", deviceCount)
	}
	if err := register(retryTx, ownerID, deviceID, "Renamed iPhone", publicKey); !errors.Is(err, ErrDeviceAccessDenied) {
		t.Fatalf("RegisterPaired(metadata conflict) error = %v, want ErrDeviceAccessDenied", err)
	}
	if err := register(retryTx, foreignOwnerID, deviceID, "Husam's iPhone", publicKey); !errors.Is(err, ErrDeviceAccessDenied) {
		t.Fatalf("RegisterPaired(owner conflict) error = %v, want ErrDeviceAccessDenied", err)
	}
	alternateKeyID := newIntegrationDeviceID(t)
	alternateKeyBytes := canonicalDeviceIntegrationPublicKey(alternateKeyID)
	alternateKey, err := cryptox.ParseX25519PublicKey(alternateKeyBytes[:])
	if err != nil {
		t.Fatalf("ParseX25519PublicKey(alternate) error = %v", err)
	}
	if err := register(retryTx, ownerID, deviceID, "Husam's iPhone", alternateKey); !errors.Is(err, ErrDeviceAccessDenied) {
		t.Fatalf("RegisterPaired(id conflict) error = %v, want ErrDeviceAccessDenied", err)
	}
	if err := register(retryTx, ownerID, newIntegrationDeviceID(t), "Second iPhone", publicKey); !errors.Is(err, ErrDeviceAccessDenied) {
		t.Fatalf("RegisterPaired(key conflict) error = %v, want ErrDeviceAccessDenied", err)
	}
	if err := register(retryTx, foreignOwnerID, newIntegrationDeviceID(t), "Foreign iPhone", publicKey); !errors.Is(err, ErrDeviceAccessDenied) {
		t.Fatalf("RegisterPaired(owner and key conflict) error = %v, want ErrDeviceAccessDenied", err)
	}
	crossedDeviceID := newIntegrationDeviceID(t)
	crossedFingerprintID := newIntegrationDeviceID(t)
	insertDeviceFixture(t, ctx, retryTx, crossedDeviceID, ownerID.String(), pairedAt)
	insertDeviceFixture(t, ctx, retryTx, crossedFingerprintID, ownerID.String(), pairedAt)
	crossedKeyBytes := canonicalDeviceIntegrationPublicKey(crossedFingerprintID)
	crossedKey, err := cryptox.ParseX25519PublicKey(crossedKeyBytes[:])
	if err != nil {
		t.Fatalf("ParseX25519PublicKey(crossed) error = %v", err)
	}
	if err := register(retryTx, ownerID, crossedDeviceID, "Integration device", crossedKey); !errors.Is(err, ErrDeviceAccessDenied) {
		t.Fatalf("RegisterPaired(crossed collision) error = %v, want ErrDeviceAccessDenied", err)
	}
	if _, err := retryTx.Exec(ctx, `UPDATE device.devices SET revoked_at = $1 WHERE id = $2`, pairedAt.Add(time.Minute), deviceID); err != nil {
		t.Fatalf("revoke paired device fixture: %v", err)
	}
	if err := register(retryTx, ownerID, deviceID, "Husam's iPhone", publicKey); !errors.Is(err, ErrDeviceAccessDenied) {
		t.Fatalf("RegisterPaired(revoked identity) error = %v, want ErrDeviceAccessDenied", err)
	}
	if err := retryTx.Rollback(ctx); err != nil {
		t.Fatalf("rollback device retry transaction: %v", err)
	}

	rollbackID := newIntegrationDeviceID(t)
	rollbackKeyBytes := canonicalDeviceIntegrationPublicKey(rollbackID)
	rollbackKey, err := cryptox.ParseX25519PublicKey(rollbackKeyBytes[:])
	if err != nil {
		t.Fatalf("ParseX25519PublicKey(rollback) error = %v", err)
	}
	rollbackTx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin device rollback transaction: %v", err)
	}
	cleanupDeviceRegistrationTransaction(t, rollbackTx)
	if err := register(rollbackTx, ownerID, rollbackID, "Rollback iPhone", rollbackKey); err != nil {
		t.Fatalf("RegisterPaired(rollback candidate) error = %v", err)
	}
	if err := rollbackTx.Rollback(ctx); err != nil {
		t.Fatalf("rollback device registration: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM device.devices WHERE id = $1`, rollbackID).Scan(&deviceCount); err != nil {
		t.Fatalf("count rolled-back paired devices: %v", err)
	}
	if deviceCount != 0 {
		t.Fatalf("rolled-back paired device count = %d, want 0", deviceCount)
	}
}

func TestRepositoryRegisterPairedTimeoutIntegration(t *testing.T) {
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

	lockingTx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin device registration locking transaction: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if err := lockingTx.Rollback(cleanupCtx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			t.Errorf("rollback device registration locking transaction: %v", err)
		}
	})
	if _, err := lockingTx.Exec(ctx, `LOCK TABLE device.devices IN ACCESS EXCLUSIVE MODE`); err != nil {
		t.Fatalf("lock device table: %v", err)
	}

	ownerID := parseDeviceRegistrationOwner(t, "0123456789abcdef0123456789abcdef")
	pairedAt := time.Date(2026, time.August, 3, 12, 0, 0, 0, time.UTC)
	deviceID := newIntegrationDeviceID(t)
	keyBytes := canonicalDeviceIntegrationPublicKey(deviceID)
	publicKey, err := cryptox.ParseX25519PublicKey(keyBytes[:])
	if err != nil {
		t.Fatalf("ParseX25519PublicKey() error = %v", err)
	}
	repository, err := NewRepository(pool, func() time.Time { return pairedAt.Add(time.Hour) })
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}
	repository.operationMax = 100 * time.Millisecond
	registrationTx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin bounded device registration transaction: %v", err)
	}
	cleanupDeviceRegistrationTransaction(t, registrationTx)
	if err := repository.RegisterPaired(
		context.Background(), registrationTx, ownerID, deviceID, "Timeout iPhone",
		PlatformIOS, publicKey, pairedAt,
	); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("RegisterPaired(locked table) error = %v, want context.DeadlineExceeded", err)
	}
	if err := registrationTx.Rollback(ctx); err != nil &&
		!errors.Is(err, pgx.ErrTxClosed) && !registrationTx.Conn().IsClosed() {
		t.Fatalf("rollback bounded device registration transaction: %v", err)
	}
	if err := lockingTx.Rollback(ctx); err != nil {
		t.Fatalf("release device table lock: %v", err)
	}
	var deviceCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM device.devices WHERE id = $1`, deviceID).Scan(&deviceCount); err != nil {
		t.Fatalf("count timed-out paired devices: %v", err)
	}
	if deviceCount != 0 {
		t.Fatalf("timed-out paired device count = %d, want 0", deviceCount)
	}

	retryTx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin device registration retry transaction: %v", err)
	}
	cleanupDeviceRegistrationTransaction(t, retryTx)
	if err := repository.RegisterPaired(
		ctx, retryTx, ownerID, deviceID, "Timeout iPhone", PlatformIOS, publicKey, pairedAt,
	); err != nil {
		t.Fatalf("RegisterPaired(after lock release) error = %v", err)
	}
	if err := retryTx.Commit(ctx); err != nil {
		t.Fatalf("commit device registration retry: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if _, err := pool.Exec(cleanupCtx, `DELETE FROM device.devices WHERE id = $1`, deviceID); err != nil {
			t.Errorf("delete timeout paired device: %v", err)
		}
	})
}

func TestRepositoryRegisterPairedConcurrentRetryIntegration(t *testing.T) {
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

	ownerID := parseDeviceRegistrationOwner(t, "0123456789abcdef0123456789abcdef")
	pairedAt := time.Date(2026, time.August, 3, 13, 0, 0, 987654321, time.UTC)
	deviceID := newIntegrationDeviceID(t)
	keyBytes := canonicalDeviceIntegrationPublicKey(deviceID)
	publicKey, err := cryptox.ParseX25519PublicKey(keyBytes[:])
	if err != nil {
		t.Fatalf("ParseX25519PublicKey() error = %v", err)
	}
	repository, err := NewRepository(pool, func() time.Time { return pairedAt.Add(time.Hour) })
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}

	firstTx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin first concurrent device transaction: %v", err)
	}
	cleanupDeviceRegistrationTransaction(t, firstTx)
	secondTx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin second concurrent device transaction: %v", err)
	}
	cleanupDeviceRegistrationTransaction(t, secondTx)
	if err := repository.RegisterPaired(
		ctx, firstTx, ownerID, deviceID, "Concurrent iPhone", PlatformIOS, publicKey, pairedAt,
	); err != nil {
		t.Fatalf("RegisterPaired(first concurrent request) error = %v", err)
	}
	var secondBackendPID int
	if err := secondTx.QueryRow(ctx, `SELECT pg_backend_pid()`).Scan(&secondBackendPID); err != nil {
		t.Fatalf("read second device backend PID: %v", err)
	}
	registrationCtx, cancelRegistration := context.WithCancel(ctx)
	secondResult := make(chan error, 1)
	secondFinished := make(chan struct{})
	go func() {
		defer close(secondFinished)
		secondResult <- repository.RegisterPaired(
			registrationCtx, secondTx, ownerID, deviceID, "Concurrent iPhone", PlatformIOS, publicKey, pairedAt,
		)
	}()
	t.Cleanup(func() {
		cancelRegistration()
		select {
		case <-secondFinished:
		case <-time.After(5 * time.Second):
			t.Error("concurrent device registration did not stop during cleanup")
		}
	})
	for {
		select {
		case err := <-secondResult:
			t.Fatalf("second RegisterPaired completed before first commit: %v", err)
		default:
		}
		var blockerCount int
		if err := pool.QueryRow(ctx, `SELECT cardinality(pg_blocking_pids($1))`, secondBackendPID).Scan(&blockerCount); err != nil {
			t.Fatalf("read blocked device registration: %v", err)
		}
		if blockerCount > 0 {
			break
		}
	}
	if err := firstTx.Commit(ctx); err != nil {
		t.Fatalf("commit first concurrent device registration: %v", err)
	}
	select {
	case err := <-secondResult:
		if err != nil {
			t.Fatalf("RegisterPaired(concurrent exact retry) error = %v", err)
		}
	case <-ctx.Done():
		t.Fatalf("wait for concurrent device retry: %v", ctx.Err())
	}
	if err := secondTx.Commit(ctx); err != nil {
		t.Fatalf("commit second concurrent device registration: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if _, err := pool.Exec(cleanupCtx, `DELETE FROM device.devices WHERE id = $1`, deviceID); err != nil {
			t.Errorf("delete concurrent paired device: %v", err)
		}
	})
	var deviceCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM device.devices WHERE id = $1`, deviceID).Scan(&deviceCount); err != nil {
		t.Fatalf("count concurrent paired devices: %v", err)
	}
	if deviceCount != 1 {
		t.Fatalf("concurrent paired device count = %d, want 1", deviceCount)
	}
}

func cleanupDeviceRegistrationTransaction(t *testing.T, tx pgx.Tx) {
	t.Helper()
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if err := tx.Rollback(cleanupCtx); err != nil &&
			!errors.Is(err, pgx.ErrTxClosed) && !tx.Conn().IsClosed() {
			t.Errorf("rollback device registration test transaction: %v", err)
		}
	})
}

func parseDeviceRegistrationOwner(t *testing.T, encoded string) auth.UserID {
	t.Helper()
	ownerID, err := auth.ParseUserID(encoded)
	if err != nil {
		t.Fatalf("ParseUserID() error = %v", err)
	}
	return ownerID
}
