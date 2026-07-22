package workspace

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/hgunduzoglu/coderoam/packages/go/ids"
	"github.com/hgunduzoglu/coderoam/packages/go/postgresx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRepositoryAuthorizeAgentIntegration(t *testing.T) {
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
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin agent authorization transaction: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if err := tx.Rollback(cleanupCtx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			t.Errorf("rollback agent authorization transaction: %v", err)
		}
	})

	owner := newWorkspaceTestActor(t, "0123456789abcdef0123456789abcdef", "owner@example.com")
	foreignActor := newWorkspaceTestActor(t, "3123456789abcdef0123456789abcdef", "foreign@example.com")
	ownerID, _ := owner.UserID()
	checkedAt := time.Date(2026, time.July, 18, 20, 0, 0, 0, time.UTC)
	repository, err := NewRepository(tx, func() time.Time { return checkedAt })
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}

	activeAgentID := newWorkspaceIntegrationID(t)
	insertAgentFixture(t, ctx, tx, activeAgentID, ownerID.String(), checkedAt.Add(-time.Hour), 0x41)
	if err := repository.AuthorizeAgent(ctx, tx, owner, activeAgentID); err != nil {
		t.Fatalf("AuthorizeAgent(owner active) error = %v", err)
	}
	assertAgentState(t, ctx, tx, activeAgentID, nil, nil)
	if err := repository.AuthorizeAgent(ctx, tx, foreignActor, activeAgentID); !errors.Is(err, ErrAgentAccessDenied) {
		t.Fatalf("AuthorizeAgent(foreign) error = %v, want ErrAgentAccessDenied", err)
	}
	if err := repository.AuthorizeAgent(ctx, tx, owner, newWorkspaceIntegrationID(t)); !errors.Is(err, ErrAgentAccessDenied) {
		t.Fatalf("AuthorizeAgent(missing) error = %v, want ErrAgentAccessDenied", err)
	}

	revokedAgentID := newWorkspaceIntegrationID(t)
	insertAgentFixture(t, ctx, tx, revokedAgentID, ownerID.String(), checkedAt.Add(-time.Hour), 0x42)
	if _, err := tx.Exec(ctx, `UPDATE workspace.agents SET revoked_at = $1 WHERE id = $2`, checkedAt, revokedAgentID); err != nil {
		t.Fatalf("revoke agent fixture: %v", err)
	}
	if err := repository.AuthorizeAgent(ctx, tx, owner, revokedAgentID); !errors.Is(err, ErrAgentAccessDenied) {
		t.Fatalf("AuthorizeAgent(revoked) error = %v, want ErrAgentAccessDenied", err)
	}

	futureAgentID := newWorkspaceIntegrationID(t)
	insertAgentFixture(t, ctx, tx, futureAgentID, ownerID.String(), checkedAt.Add(time.Hour), 0x43)
	if err := repository.AuthorizeAgent(ctx, tx, owner, futureAgentID); !errors.Is(err, ErrAgentAccessDenied) {
		t.Fatalf("AuthorizeAgent(future creation) error = %v, want ErrAgentAccessDenied", err)
	}

	corruptKeyAgentID := newWorkspaceIntegrationID(t)
	insertAgentFixture(t, ctx, tx, corruptKeyAgentID, ownerID.String(), checkedAt.Add(-time.Hour), 0x44)
	if _, err := tx.Exec(ctx, `UPDATE workspace.agents SET static_public_key = $1 WHERE id = $2`, []byte{0x44}, corruptKeyAgentID); err != nil {
		t.Fatalf("corrupt stored agent public key: %v", err)
	}
	if err := repository.AuthorizeAgent(ctx, tx, owner, corruptKeyAgentID); !errors.Is(err, ErrAgentAccessDenied) {
		t.Fatalf("AuthorizeAgent(corrupt key) error = %v, want ErrAgentAccessDenied", err)
	}

	corruptVersionAgentID := newWorkspaceIntegrationID(t)
	insertAgentFixture(t, ctx, tx, corruptVersionAgentID, ownerID.String(), checkedAt.Add(-time.Hour), 0x45)
	if _, err := tx.Exec(ctx, `UPDATE workspace.agents SET version = $1 WHERE id = $2`, "0.1.0\nforged", corruptVersionAgentID); err != nil {
		t.Fatalf("corrupt stored agent version: %v", err)
	}
	if err := repository.AuthorizeAgent(ctx, tx, owner, corruptVersionAgentID); !errors.Is(err, ErrAgentAccessDenied) {
		t.Fatalf("AuthorizeAgent(corrupt version) error = %v, want ErrAgentAccessDenied", err)
	}
}

func TestRepositoryAuthorizeAgentTimeoutIntegration(t *testing.T) {
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

	owner := newWorkspaceTestActor(t, "0123456789abcdef0123456789abcdef", "owner@example.com")
	ownerID, _ := owner.UserID()
	checkedAt := time.Date(2026, time.July, 18, 20, 0, 0, 0, time.UTC)
	agentID := insertCommittedAgentFixture(t, ctx, pool, ownerID.String(), checkedAt.Add(-time.Hour), 0x46)

	lockingTx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin agent authorization locking transaction: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if err := lockingTx.Rollback(cleanupCtx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			t.Errorf("rollback agent authorization locking transaction: %v", err)
		}
	})
	if _, err := lockingTx.Exec(ctx, `LOCK TABLE workspace.agents IN ACCESS EXCLUSIVE MODE`); err != nil {
		t.Fatalf("lock workspace agent table: %v", err)
	}

	repository, err := NewRepository(pool, func() time.Time { return checkedAt })
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}
	repository.operationMax = 100 * time.Millisecond
	authorizationTx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin bounded agent authorization transaction: %v", err)
	}
	if err := repository.AuthorizeAgent(
		context.Background(), authorizationTx, owner, agentID,
	); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("AuthorizeAgent(locked table) error = %v, want context.DeadlineExceeded", err)
	}
	if err := authorizationTx.Rollback(ctx); err != nil &&
		!errors.Is(err, pgx.ErrTxClosed) && !authorizationTx.Conn().IsClosed() {
		t.Fatalf("rollback bounded agent authorization transaction: %v", err)
	}
	if err := lockingTx.Rollback(ctx); err != nil {
		t.Fatalf("release workspace agent table lock: %v", err)
	}

	retryTx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin agent authorization retry transaction: %v", err)
	}
	if err := repository.AuthorizeAgent(ctx, retryTx, owner, agentID); err != nil {
		t.Fatalf("AuthorizeAgent(after lock release) error = %v", err)
	}
	if err := retryTx.Commit(ctx); err != nil {
		t.Fatalf("commit agent authorization retry transaction: %v", err)
	}
	assertAgentState(t, ctx, pool, agentID, nil, nil)
}

func TestRepositoryAuthorizeAgentLockIntegration(t *testing.T) {
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

	owner := newWorkspaceTestActor(t, "0123456789abcdef0123456789abcdef", "owner@example.com")
	ownerID, _ := owner.UserID()
	checkedAt := time.Date(2026, time.July, 18, 20, 0, 0, 0, time.UTC)
	agentID := insertCommittedAgentFixture(t, ctx, pool, ownerID.String(), checkedAt.Add(-time.Hour), 0x47)
	repository, err := NewRepository(pool, func() time.Time { return checkedAt })
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}

	authorizationTx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin active agent authorization transaction: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if err := authorizationTx.Rollback(cleanupCtx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			t.Errorf("rollback active agent authorization transaction: %v", err)
		}
	})
	if err := repository.AuthorizeAgent(ctx, authorizationTx, owner, agentID); err != nil {
		t.Fatalf("AuthorizeAgent(active transaction) error = %v", err)
	}

	revocationTx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin blocked agent revocation transaction: %v", err)
	}
	revocationCtx, cancelRevocation := context.WithTimeout(context.Background(), 100*time.Millisecond)
	_, revocationErr := revocationTx.Exec(revocationCtx, `
		UPDATE workspace.agents SET revoked_at = $1 WHERE id = $2`, checkedAt, agentID)
	cancelRevocation()
	if !errors.Is(revocationErr, context.DeadlineExceeded) {
		t.Fatalf("revocation while authorization open error = %v, want context.DeadlineExceeded", revocationErr)
	}
	if err := revocationTx.Rollback(ctx); err != nil &&
		!errors.Is(err, pgx.ErrTxClosed) && !revocationTx.Conn().IsClosed() {
		t.Fatalf("rollback blocked agent revocation transaction: %v", err)
	}
	assertAgentState(t, ctx, pool, agentID, nil, nil)

	if err := authorizationTx.Commit(ctx); err != nil {
		t.Fatalf("commit active agent authorization transaction: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE workspace.agents SET revoked_at = $1 WHERE id = $2`, checkedAt, agentID); err != nil {
		t.Fatalf("revoke agent after authorization commit: %v", err)
	}
	assertAgentState(t, ctx, pool, agentID, nil, &checkedAt)

	deniedTx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin revoked-agent authorization transaction: %v", err)
	}
	if err := repository.AuthorizeAgent(ctx, deniedTx, owner, agentID); !errors.Is(err, ErrAgentAccessDenied) {
		t.Fatalf("AuthorizeAgent(after revocation) error = %v, want ErrAgentAccessDenied", err)
	}
	if err := deniedTx.Rollback(ctx); err != nil {
		t.Fatalf("rollback revoked-agent authorization transaction: %v", err)
	}
}

func applyWorkspaceIntegrationMigrations(
	t *testing.T,
	ctx context.Context,
	database postgresx.TransactionStarter,
) {
	t.Helper()
	for _, migration := range []struct {
		scope string
		path  string
	}{
		{scope: "workspace", path: "migrations/000001_init.sql"},
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

func insertAgentFixture(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	agentID string,
	ownerID string,
	createdAt time.Time,
	keyByte byte,
) {
	t.Helper()
	if _, err := tx.Exec(ctx, `
		INSERT INTO workspace.agents (
			id, user_id, name, static_public_key, public_key_fingerprint, version, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		agentID,
		ownerID,
		"Integration workspace agent",
		bytes.Repeat([]byte{keyByte}, 32),
		"fixture:"+agentID,
		"0.1.0",
		createdAt,
	); err != nil {
		t.Fatalf("insert workspace agent fixture: %v", err)
	}
}

func insertCommittedAgentFixture(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	ownerID string,
	createdAt time.Time,
	keyByte byte,
) string {
	t.Helper()
	agentID := newWorkspaceIntegrationID(t)
	if _, err := pool.Exec(ctx, `
		INSERT INTO workspace.agents (
			id, user_id, name, static_public_key, public_key_fingerprint, version, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		agentID,
		ownerID,
		"Committed workspace agent",
		bytes.Repeat([]byte{keyByte}, 32),
		"fixture:"+agentID,
		"0.1.0",
		createdAt,
	); err != nil {
		t.Fatalf("insert committed workspace agent fixture: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if _, err := pool.Exec(cleanupCtx, `DELETE FROM workspace.agents WHERE id = $1`, agentID); err != nil {
			t.Errorf("delete committed workspace agent fixture: %v", err)
		}
	})
	return agentID
}

type workspaceAgentStateReader interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func assertAgentState(
	t *testing.T,
	ctx context.Context,
	reader workspaceAgentStateReader,
	agentID string,
	wantLastSeenAt *time.Time,
	wantRevokedAt *time.Time,
) {
	t.Helper()
	var lastSeenAt, revokedAt *time.Time
	if err := reader.QueryRow(ctx, `
		SELECT last_seen_at, revoked_at FROM workspace.agents WHERE id = $1`, agentID).Scan(
		&lastSeenAt, &revokedAt,
	); err != nil {
		t.Fatalf("read workspace agent state: %v", err)
	}
	if !equalOptionalTime(lastSeenAt, wantLastSeenAt) {
		t.Fatalf("last_seen_at = %v, want %v", lastSeenAt, wantLastSeenAt)
	}
	if !equalOptionalTime(revokedAt, wantRevokedAt) {
		t.Fatalf("revoked_at = %v, want %v", revokedAt, wantRevokedAt)
	}
}

func equalOptionalTime(got *time.Time, want *time.Time) bool {
	if got == nil || want == nil {
		return got == nil && want == nil
	}
	return got.Equal(*want)
}

func newWorkspaceIntegrationID(t *testing.T) string {
	t.Helper()
	id, err := ids.New()
	if err != nil {
		t.Fatalf("ids.New() error = %v", err)
	}
	return id.String()
}
