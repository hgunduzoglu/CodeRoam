package workspace

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

func TestRepositoryRegisterPairedAgentIntegration(t *testing.T) {
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
	applyWorkspaceIntegrationMigrations(t, ctx, pool)
	firstTx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin agent registration transaction: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if err := firstTx.Rollback(cleanupCtx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			t.Errorf("rollback first agent registration transaction: %v", err)
		}
	})

	ownerID := parseAgentRegistrationOwner(t, "0123456789abcdef0123456789abcdef")
	foreignOwnerID := parseAgentRegistrationOwner(t, "2123456789abcdef0123456789abcdef")
	createdAt := time.Date(2026, time.August, 3, 11, 0, 0, 123456789, time.UTC)
	repository, err := NewRepository(pool, func() time.Time { return createdAt.Add(time.Hour) })
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}
	agentID := newWorkspaceIntegrationID(t)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if _, err := pool.Exec(cleanupCtx, `DELETE FROM workspace.agents WHERE id = $1`, agentID); err != nil {
			t.Errorf("delete committed paired agent: %v", err)
		}
	})
	publicKey := newWorkspaceTestPublicKey(t, 0x42)

	register := func(tx pgx.Tx, owner auth.UserID, id, name string, key cryptox.X25519PublicKey) error {
		return repository.RegisterPairedAgent(
			ctx, tx, owner, id, name, key, "0.1.0", createdAt,
		)
	}
	if err := register(firstTx, ownerID, agentID, "MacBook Agent", publicKey); err != nil {
		t.Fatalf("RegisterPairedAgent(new) error = %v", err)
	}
	if err := firstTx.Commit(ctx); err != nil {
		t.Fatalf("commit first agent registration: %v", err)
	}

	retryTx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin agent retry transaction: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if err := retryTx.Rollback(cleanupCtx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			t.Errorf("rollback agent retry transaction: %v", err)
		}
	})
	if err := register(retryTx, ownerID, agentID, "MacBook Agent", publicKey); err != nil {
		t.Fatalf("RegisterPairedAgent(committed exact retry) error = %v", err)
	}
	var agentCount int
	if err := retryTx.QueryRow(ctx, `SELECT count(*) FROM workspace.agents WHERE id = $1`, agentID).Scan(&agentCount); err != nil {
		t.Fatalf("count paired agents: %v", err)
	}
	if agentCount != 1 {
		t.Fatalf("paired agent count = %d, want 1", agentCount)
	}
	if err := register(retryTx, ownerID, agentID, "Renamed Agent", publicKey); !errors.Is(err, ErrAgentAccessDenied) {
		t.Fatalf("RegisterPairedAgent(metadata conflict) error = %v, want ErrAgentAccessDenied", err)
	}
	if err := register(retryTx, foreignOwnerID, agentID, "MacBook Agent", publicKey); !errors.Is(err, ErrAgentAccessDenied) {
		t.Fatalf("RegisterPairedAgent(owner conflict) error = %v, want ErrAgentAccessDenied", err)
	}
	alternateKey := newWorkspaceTestPublicKey(t, 0x43)
	if err := register(retryTx, ownerID, agentID, "MacBook Agent", alternateKey); !errors.Is(err, ErrAgentAccessDenied) {
		t.Fatalf("RegisterPairedAgent(id conflict) error = %v, want ErrAgentAccessDenied", err)
	}
	if err := register(retryTx, ownerID, newWorkspaceIntegrationID(t), "Second Agent", publicKey); !errors.Is(err, ErrAgentAccessDenied) {
		t.Fatalf("RegisterPairedAgent(key conflict) error = %v, want ErrAgentAccessDenied", err)
	}
	if err := register(retryTx, foreignOwnerID, newWorkspaceIntegrationID(t), "Foreign Agent", publicKey); !errors.Is(err, ErrAgentAccessDenied) {
		t.Fatalf("RegisterPairedAgent(owner and key conflict) error = %v, want ErrAgentAccessDenied", err)
	}
	crossedAgentID := newWorkspaceIntegrationID(t)
	crossedFingerprintID := newWorkspaceIntegrationID(t)
	insertAgentFixture(t, ctx, retryTx, crossedAgentID, ownerID.String(), createdAt, 0x44)
	insertAgentFixture(t, ctx, retryTx, crossedFingerprintID, ownerID.String(), createdAt, 0x45)
	crossedKey := newWorkspaceTestPublicKey(t, 0x45)
	if err := register(retryTx, ownerID, crossedAgentID, "Integration workspace agent", crossedKey); !errors.Is(err, ErrAgentAccessDenied) {
		t.Fatalf("RegisterPairedAgent(crossed collision) error = %v, want ErrAgentAccessDenied", err)
	}
	if _, err := retryTx.Exec(ctx, `UPDATE workspace.agents SET revoked_at = $1 WHERE id = $2`, createdAt.Add(time.Minute), agentID); err != nil {
		t.Fatalf("revoke paired agent fixture: %v", err)
	}
	if err := register(retryTx, ownerID, agentID, "MacBook Agent", publicKey); !errors.Is(err, ErrAgentAccessDenied) {
		t.Fatalf("RegisterPairedAgent(revoked identity) error = %v, want ErrAgentAccessDenied", err)
	}
	if err := retryTx.Rollback(ctx); err != nil {
		t.Fatalf("rollback agent retry transaction: %v", err)
	}

	rollbackID := newWorkspaceIntegrationID(t)
	rollbackKey := newWorkspaceTestPublicKey(t, 0x46)
	rollbackTx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin agent rollback transaction: %v", err)
	}
	cleanupAgentRegistrationTransaction(t, rollbackTx)
	if err := register(rollbackTx, ownerID, rollbackID, "Rollback Agent", rollbackKey); err != nil {
		t.Fatalf("RegisterPairedAgent(rollback candidate) error = %v", err)
	}
	if err := rollbackTx.Rollback(ctx); err != nil {
		t.Fatalf("rollback agent registration: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM workspace.agents WHERE id = $1`, rollbackID).Scan(&agentCount); err != nil {
		t.Fatalf("count rolled-back paired agents: %v", err)
	}
	if agentCount != 0 {
		t.Fatalf("rolled-back paired agent count = %d, want 0", agentCount)
	}
}

func TestRepositoryRegisterPairedAgentTimeoutIntegration(t *testing.T) {
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
	applyWorkspaceIntegrationMigrations(t, ctx, pool)

	lockingTx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin agent registration locking transaction: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if err := lockingTx.Rollback(cleanupCtx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			t.Errorf("rollback agent registration locking transaction: %v", err)
		}
	})
	if _, err := lockingTx.Exec(ctx, `LOCK TABLE workspace.agents IN ACCESS EXCLUSIVE MODE`); err != nil {
		t.Fatalf("lock workspace agent table: %v", err)
	}

	ownerID := parseAgentRegistrationOwner(t, "0123456789abcdef0123456789abcdef")
	createdAt := time.Date(2026, time.August, 3, 12, 0, 0, 0, time.UTC)
	agentID := newWorkspaceIntegrationID(t)
	publicKey := newWorkspaceTestPublicKey(t, 0x47)
	repository, err := NewRepository(pool, func() time.Time { return createdAt.Add(time.Hour) })
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}
	repository.operationMax = 100 * time.Millisecond
	registrationTx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin bounded agent registration transaction: %v", err)
	}
	cleanupAgentRegistrationTransaction(t, registrationTx)
	if err := repository.RegisterPairedAgent(
		context.Background(), registrationTx, ownerID, agentID, "Timeout Agent",
		publicKey, "0.1.0", createdAt,
	); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("RegisterPairedAgent(locked table) error = %v, want context.DeadlineExceeded", err)
	}
	if err := registrationTx.Rollback(ctx); err != nil &&
		!errors.Is(err, pgx.ErrTxClosed) && !registrationTx.Conn().IsClosed() {
		t.Fatalf("rollback bounded agent registration transaction: %v", err)
	}
	if err := lockingTx.Rollback(ctx); err != nil {
		t.Fatalf("release workspace agent table lock: %v", err)
	}
	var agentCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM workspace.agents WHERE id = $1`, agentID).Scan(&agentCount); err != nil {
		t.Fatalf("count timed-out paired agents: %v", err)
	}
	if agentCount != 0 {
		t.Fatalf("timed-out paired agent count = %d, want 0", agentCount)
	}

	retryTx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin agent registration retry transaction: %v", err)
	}
	cleanupAgentRegistrationTransaction(t, retryTx)
	if err := repository.RegisterPairedAgent(
		ctx, retryTx, ownerID, agentID, "Timeout Agent", publicKey, "0.1.0", createdAt,
	); err != nil {
		t.Fatalf("RegisterPairedAgent(after lock release) error = %v", err)
	}
	if err := retryTx.Commit(ctx); err != nil {
		t.Fatalf("commit agent registration retry: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if _, err := pool.Exec(cleanupCtx, `DELETE FROM workspace.agents WHERE id = $1`, agentID); err != nil {
			t.Errorf("delete timeout paired agent: %v", err)
		}
	})
}

func TestRepositoryRegisterPairedAgentConcurrentRetryIntegration(t *testing.T) {
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
	applyWorkspaceIntegrationMigrations(t, ctx, pool)

	ownerID := parseAgentRegistrationOwner(t, "0123456789abcdef0123456789abcdef")
	createdAt := time.Date(2026, time.August, 3, 13, 0, 0, 987654321, time.UTC)
	agentID := newWorkspaceIntegrationID(t)
	publicKey := newWorkspaceTestPublicKey(t, 0x48)
	repository, err := NewRepository(pool, func() time.Time { return createdAt.Add(time.Hour) })
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}

	firstTx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin first concurrent agent transaction: %v", err)
	}
	cleanupAgentRegistrationTransaction(t, firstTx)
	secondTx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin second concurrent agent transaction: %v", err)
	}
	cleanupAgentRegistrationTransaction(t, secondTx)
	if err := repository.RegisterPairedAgent(
		ctx, firstTx, ownerID, agentID, "Concurrent Agent", publicKey, "0.1.0", createdAt,
	); err != nil {
		t.Fatalf("RegisterPairedAgent(first concurrent request) error = %v", err)
	}
	var secondBackendPID int
	if err := secondTx.QueryRow(ctx, `SELECT pg_backend_pid()`).Scan(&secondBackendPID); err != nil {
		t.Fatalf("read second agent backend PID: %v", err)
	}
	registrationCtx, cancelRegistration := context.WithCancel(ctx)
	secondResult := make(chan error, 1)
	secondFinished := make(chan struct{})
	go func() {
		defer close(secondFinished)
		secondResult <- repository.RegisterPairedAgent(
			registrationCtx, secondTx, ownerID, agentID, "Concurrent Agent", publicKey, "0.1.0", createdAt,
		)
	}()
	t.Cleanup(func() {
		cancelRegistration()
		select {
		case <-secondFinished:
		case <-time.After(5 * time.Second):
			t.Error("concurrent agent registration did not stop during cleanup")
		}
	})
	for {
		select {
		case err := <-secondResult:
			t.Fatalf("second RegisterPairedAgent completed before first commit: %v", err)
		default:
		}
		var blockerCount int
		if err := pool.QueryRow(ctx, `SELECT cardinality(pg_blocking_pids($1))`, secondBackendPID).Scan(&blockerCount); err != nil {
			t.Fatalf("read blocked agent registration: %v", err)
		}
		if blockerCount > 0 {
			break
		}
	}
	if err := firstTx.Commit(ctx); err != nil {
		t.Fatalf("commit first concurrent agent registration: %v", err)
	}
	select {
	case err := <-secondResult:
		if err != nil {
			t.Fatalf("RegisterPairedAgent(concurrent exact retry) error = %v", err)
		}
	case <-ctx.Done():
		t.Fatalf("wait for concurrent agent retry: %v", ctx.Err())
	}
	if err := secondTx.Commit(ctx); err != nil {
		t.Fatalf("commit second concurrent agent registration: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if _, err := pool.Exec(cleanupCtx, `DELETE FROM workspace.agents WHERE id = $1`, agentID); err != nil {
			t.Errorf("delete concurrent paired agent: %v", err)
		}
	})
	var agentCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM workspace.agents WHERE id = $1`, agentID).Scan(&agentCount); err != nil {
		t.Fatalf("count concurrent paired agents: %v", err)
	}
	if agentCount != 1 {
		t.Fatalf("concurrent paired agent count = %d, want 1", agentCount)
	}
}

func cleanupAgentRegistrationTransaction(t *testing.T, tx pgx.Tx) {
	t.Helper()
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if err := tx.Rollback(cleanupCtx); err != nil &&
			!errors.Is(err, pgx.ErrTxClosed) && !tx.Conn().IsClosed() {
			t.Errorf("rollback agent registration test transaction: %v", err)
		}
	})
}

func parseAgentRegistrationOwner(t *testing.T, encoded string) auth.UserID {
	t.Helper()
	ownerID, err := auth.ParseUserID(encoded)
	if err != nil {
		t.Fatalf("ParseUserID() error = %v", err)
	}
	return ownerID
}
