package session

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/hgunduzoglu/coderoam/packages/go/ids"
	"github.com/hgunduzoglu/coderoam/packages/go/postgresx"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/auth"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRepositoryCreateIntegration(t *testing.T) {
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
	applySessionIntegrationMigration(t, ctx, pool)
	repository := NewRepository()
	owner := newSessionTestActor(t, "0123456789abcdef0123456789abcdef")
	startedAt := time.Date(2026, time.July, 19, 13, 30, 0, 0, time.UTC)

	rolledBack := newSessionIntegrationMetadata(t, owner, startedAt)
	rollbackTx := beginSessionIntegrationTx(t, ctx, pool, "rollback")
	if err := repository.Create(ctx, rollbackTx, rolledBack); err != nil {
		t.Fatalf("Create(rollback) error = %v", err)
	}
	assertStoredSession(t, ctx, rollbackTx, rolledBack)
	assertSessionMissing(t, ctx, pool, rolledBack.id.String())
	if err := rollbackTx.Rollback(ctx); err != nil {
		t.Fatalf("rollback created session: %v", err)
	}
	assertSessionMissing(t, ctx, pool, rolledBack.id.String())

	committed := newSessionIntegrationMetadata(t, owner, startedAt.Add(time.Minute))
	assertSessionMissing(t, ctx, pool, committed.id.String())
	deleteSessionIntegrationFixture(t, pool, committed.id.String())
	commitTx := beginSessionIntegrationTx(t, ctx, pool, "commit")
	if err := repository.Create(ctx, commitTx, committed); err != nil {
		t.Fatalf("Create(commit) error = %v", err)
	}
	assertSessionMissing(t, ctx, pool, committed.id.String())
	if err := commitTx.Commit(ctx); err != nil {
		t.Fatalf("commit created session: %v", err)
	}
	assertStoredSession(t, ctx, pool, committed)

	duplicateTx := beginSessionIntegrationTx(t, ctx, pool, "duplicate")
	if err := repository.Create(ctx, duplicateTx, committed); !errors.Is(err, ErrSessionAlreadyExists) {
		t.Fatalf("Create(duplicate) error = %v, want ErrSessionAlreadyExists", err)
	}
	rollbackSessionIntegrationTx(t, duplicateTx, "duplicate")

	invalidTx := beginSessionIntegrationTx(t, ctx, pool, "invalid")
	if err := repository.Create(ctx, invalidTx, Session{}); !errors.Is(err, ErrInvalidSession) {
		t.Fatalf("Create(zero session) error = %v, want ErrInvalidSession", err)
	}
	canceledCtx, cancelCreate := context.WithCancel(context.Background())
	cancelCreate()
	if err := repository.Create(canceledCtx, invalidTx, committed); !errors.Is(err, context.Canceled) {
		t.Fatalf("Create(canceled) error = %v, want context.Canceled", err)
	}
	if _, err := invalidTx.Exec(ctx, `SELECT 1`); err != nil {
		t.Fatalf("invalid boundaries made caller transaction unusable: %v", err)
	}
	rollbackSessionIntegrationTx(t, invalidTx, "invalid")

	contended := newSessionIntegrationMetadata(t, owner, startedAt.Add(2*time.Minute))
	assertSessionMissing(t, ctx, pool, contended.id.String())
	deleteSessionIntegrationFixture(t, pool, contended.id.String())
	lockingTx := beginSessionIntegrationTx(t, ctx, pool, "locking")
	if err := repository.Create(ctx, lockingTx, contended); err != nil {
		t.Fatalf("Create(locking) error = %v", err)
	}
	boundedTx := beginSessionIntegrationTx(t, ctx, pool, "bounded")
	repository.operationMax = 100 * time.Millisecond
	if err := repository.Create(
		context.Background(), boundedTx, contended,
	); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Create(contended) error = %v, want context.DeadlineExceeded", err)
	}
	rollbackSessionIntegrationTx(t, boundedTx, "bounded")
	if err := lockingTx.Rollback(ctx); err != nil {
		t.Fatalf("release uncommitted session ID: %v", err)
	}

	retryTx := beginSessionIntegrationTx(t, ctx, pool, "retry")
	if err := repository.Create(ctx, retryTx, contended); err != nil {
		t.Fatalf("Create(after contention) error = %v", err)
	}
	if err := retryTx.Commit(ctx); err != nil {
		t.Fatalf("commit session after contention: %v", err)
	}
	assertStoredSession(t, ctx, pool, contended)
}

func applySessionIntegrationMigration(
	t *testing.T,
	ctx context.Context,
	database postgresx.TransactionStarter,
) {
	t.Helper()
	sql, err := os.ReadFile("migrations/000001_init.sql")
	if err != nil {
		t.Fatalf("read session migration: %v", err)
	}
	if err := postgresx.ApplyMigrations(ctx, database, []postgresx.Migration{{
		Scope: "session", Version: 1, Name: "init", SQL: string(sql),
	}}); err != nil {
		t.Fatalf("apply session migration: %v", err)
	}
}

func newSessionIntegrationMetadata(t *testing.T, owner auth.Actor, startedAt time.Time) Session {
	t.Helper()
	values := make([]string, 4)
	for index := range values {
		value, err := ids.New()
		if err != nil {
			t.Fatalf("ids.New() error = %v", err)
		}
		values[index] = value.String()
	}
	metadata, err := NewSession(
		owner, values[0], values[1], values[2], values[3], "eu-central-1", startedAt,
	)
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}
	return metadata
}

func beginSessionIntegrationTx(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	name string,
) pgx.Tx {
	t.Helper()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin %s session transaction: %v", name, err)
	}
	t.Cleanup(func() {
		rollbackSessionIntegrationTx(t, tx, name)
	})
	return tx
}

func rollbackSessionIntegrationTx(t *testing.T, tx pgx.Tx, name string) {
	t.Helper()
	cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cleanupCancel()
	if err := tx.Rollback(cleanupCtx); err != nil && !errors.Is(err, pgx.ErrTxClosed) && !tx.Conn().IsClosed() {
		t.Fatalf("rollback %s session transaction: %v", name, err)
	}
}

type sessionRowReader interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func assertStoredSession(t *testing.T, ctx context.Context, reader sessionRowReader, want Session) {
	t.Helper()
	var id, ownerID, deviceID, agentID, projectID, relayRegion string
	var startedAt time.Time
	var endedAt *time.Time
	var result *string
	if err := reader.QueryRow(ctx, `
		SELECT id, user_id, device_id, agent_id, project_id, relay_region, started_at, ended_at, result
		FROM session.sessions WHERE id = $1`, want.id.String(),
	).Scan(
		&id, &ownerID, &deviceID, &agentID, &projectID, &relayRegion, &startedAt, &endedAt, &result,
	); err != nil {
		t.Fatalf("read stored session: %v", err)
	}
	if id != want.id.String() || ownerID != want.ownerID.String() || deviceID != want.deviceID.String() ||
		agentID != want.agentID.String() || projectID != want.projectID.String() ||
		relayRegion != want.relayRegion || !startedAt.Equal(want.startedAt) || endedAt != nil || result != nil {
		t.Fatal("stored session did not preserve the metadata-only contract")
	}
}

func assertSessionMissing(t *testing.T, ctx context.Context, reader sessionRowReader, sessionID string) {
	t.Helper()
	var storedID string
	err := reader.QueryRow(ctx, `SELECT id FROM session.sessions WHERE id = $1`, sessionID).Scan(&storedID)
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("session %s lookup error = %v, want pgx.ErrNoRows", sessionID, err)
	}
}

func deleteSessionIntegrationFixture(t *testing.T, pool *pgxpool.Pool, sessionID string) {
	t.Helper()
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if _, err := pool.Exec(cleanupCtx, `DELETE FROM session.sessions WHERE id = $1`, sessionID); err != nil {
			t.Errorf("delete session fixture: %v", err)
		}
	})
}
