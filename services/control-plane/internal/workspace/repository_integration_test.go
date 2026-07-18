package workspace

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/hgunduzoglu/coderoam/packages/go/postgresx"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/outbox"
	"github.com/jackc/pgx/v5"
)

type ambiguousCommitStarter struct {
	transactions   transactionStarter
	afterCommitErr error
}

func (starter *ambiguousCommitStarter) Begin(ctx context.Context) (pgx.Tx, error) {
	tx, err := starter.transactions.Begin(ctx)
	if err != nil {
		return nil, err
	}
	afterCommitErr := starter.afterCommitErr
	starter.afterCommitErr = nil
	return &ambiguousCommitTx{Tx: tx, afterCommitErr: afterCommitErr}, nil
}

type ambiguousCommitTx struct {
	pgx.Tx
	afterCommitErr error
}

func (tx *ambiguousCommitTx) Commit(ctx context.Context) error {
	if err := tx.Tx.Commit(ctx); err != nil {
		return err
	}
	return tx.afterCommitErr
}

func TestRepositoryAgentRevocationIntegration(t *testing.T) {
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
		t.Fatalf("begin workspace repository transaction: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if err := tx.Rollback(cleanupCtx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			t.Errorf("rollback workspace repository transaction: %v", err)
		}
	})

	owner := newWorkspaceTestActor(t, "0123456789abcdef0123456789abcdef", "owner@example.com")
	foreignActor := newWorkspaceTestActor(t, "3123456789abcdef0123456789abcdef", "foreign@example.com")
	ownerID, _ := owner.UserID()
	revokedAt := time.Date(2026, time.July, 19, 9, 0, 0, 0, time.UTC)
	createdAt := revokedAt.Add(-time.Hour)
	repository, err := NewRepository(tx, func() time.Time { return revokedAt })
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}

	ownedAgentID := newWorkspaceIntegrationID(t)
	insertAgentFixture(t, ctx, tx, ownedAgentID, ownerID.String(), createdAt, 0x51)
	if err := repository.RevokeAgent(ctx, owner, ownedAgentID); err != nil {
		t.Fatalf("RevokeAgent(owner) error = %v", err)
	}
	assertAgentRevocation(t, ctx, tx, ownedAgentID, &revokedAt, 1)
	var eventType, aggregateType, payload string
	var availableAt time.Time
	if err := tx.QueryRow(ctx, `
		SELECT event_type, aggregate_type, payload::text, available_at
		FROM outbox.events
		WHERE aggregate_id = $1`, ownedAgentID).Scan(
		&eventType, &aggregateType, &payload, &availableAt,
	); err != nil {
		t.Fatalf("read agent revocation event: %v", err)
	}
	if eventType != "agent.revoked.v1" || aggregateType != "agent" || payload != "{}" ||
		!availableAt.Equal(revokedAt) {
		t.Fatal("agent revocation did not preserve the metadata-only outbox contract")
	}
	if err := repository.RevokeAgent(ctx, owner, ownedAgentID); err != nil {
		t.Fatalf("RevokeAgent(owner repeated) error = %v", err)
	}
	assertAgentRevocation(t, ctx, tx, ownedAgentID, &revokedAt, 1)

	ambiguousAgentID := newWorkspaceIntegrationID(t)
	insertAgentFixture(t, ctx, tx, ambiguousAgentID, ownerID.String(), createdAt, 0x57)
	ambiguousRepository, err := NewRepository(&ambiguousCommitStarter{
		transactions:   tx,
		afterCommitErr: errors.New("forced ambiguous commit result"),
	}, func() time.Time { return revokedAt })
	if err != nil {
		t.Fatalf("NewRepository(ambiguous commit) error = %v", err)
	}
	if err := ambiguousRepository.RevokeAgent(
		ctx, owner, ambiguousAgentID,
	); !errors.Is(err, ErrWorkspacePersistenceUnavailable) {
		t.Fatalf("RevokeAgent(ambiguous commit) error = %v, want ErrWorkspacePersistenceUnavailable", err)
	}
	assertAgentRevocation(t, ctx, tx, ambiguousAgentID, &revokedAt, 1)
	if err := ambiguousRepository.RevokeAgent(ctx, owner, ambiguousAgentID); err != nil {
		t.Fatalf("RevokeAgent(after ambiguous commit) error = %v", err)
	}
	assertAgentRevocation(t, ctx, tx, ambiguousAgentID, &revokedAt, 1)

	foreignAgentID := newWorkspaceIntegrationID(t)
	insertAgentFixture(t, ctx, tx, foreignAgentID, ownerID.String(), createdAt, 0x52)
	if err := repository.RevokeAgent(ctx, foreignActor, foreignAgentID); !errors.Is(err, ErrAgentAccessDenied) {
		t.Fatalf("RevokeAgent(foreign) error = %v, want ErrAgentAccessDenied", err)
	}
	assertAgentRevocation(t, ctx, tx, foreignAgentID, nil, 0)
	if err := repository.RevokeAgent(
		ctx, owner, newWorkspaceIntegrationID(t),
	); !errors.Is(err, ErrAgentAccessDenied) {
		t.Fatalf("RevokeAgent(missing) error = %v, want ErrAgentAccessDenied", err)
	}

	corruptOwnerAgentID := newWorkspaceIntegrationID(t)
	insertAgentFixture(t, ctx, tx, corruptOwnerAgentID, "not-an-owner-id", createdAt, 0x53)
	if err := repository.RevokeAgent(
		ctx, owner, corruptOwnerAgentID,
	); !errors.Is(err, ErrAgentAccessDenied) {
		t.Fatalf("RevokeAgent(corrupt owner) error = %v, want ErrAgentAccessDenied", err)
	}
	assertAgentRevocation(t, ctx, tx, corruptOwnerAgentID, nil, 0)

	futureAgentID := newWorkspaceIntegrationID(t)
	insertAgentFixture(t, ctx, tx, futureAgentID, ownerID.String(), revokedAt.Add(time.Hour), 0x54)
	if err := repository.RevokeAgent(ctx, owner, futureAgentID); !errors.Is(err, ErrInvalidAgent) {
		t.Fatalf("RevokeAgent(future creation) error = %v, want ErrInvalidAgent", err)
	}
	assertAgentRevocation(t, ctx, tx, futureAgentID, nil, 0)

	recoveryAgentID := newWorkspaceIntegrationID(t)
	insertAgentFixture(t, ctx, tx, recoveryAgentID, ownerID.String(), createdAt, 0x55)
	repository.enqueue = func(context.Context, pgx.Tx, outbox.Event) error {
		return errors.New("forced outbox failure")
	}
	if err := repository.RevokeAgent(
		ctx, owner, recoveryAgentID,
	); !errors.Is(err, ErrWorkspacePersistenceUnavailable) {
		t.Fatalf("RevokeAgent(outbox failure) error = %v, want ErrWorkspacePersistenceUnavailable", err)
	}
	assertAgentRevocation(t, ctx, tx, recoveryAgentID, nil, 0)
	repository.enqueue = outbox.Enqueue
	if err := repository.RevokeAgent(ctx, owner, recoveryAgentID); err != nil {
		t.Fatalf("RevokeAgent(recovery) error = %v", err)
	}
	assertAgentRevocation(t, ctx, tx, recoveryAgentID, &revokedAt, 1)

	lockedAgentID := insertCommittedAgentFixture(
		t, ctx, pool, ownerID.String(), createdAt, 0x56,
	)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if _, err := pool.Exec(
			cleanupCtx,
			`DELETE FROM outbox.events WHERE aggregate_type = 'agent' AND aggregate_id = $1`,
			lockedAgentID,
		); err != nil {
			t.Errorf("delete locked-agent outbox fixture: %v", err)
		}
	})
	lockingTx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin agent locking transaction: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if err := lockingTx.Rollback(cleanupCtx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			t.Errorf("rollback agent locking transaction: %v", err)
		}
	})
	var selectedAgentID string
	if err := lockingTx.QueryRow(ctx, `
		SELECT id FROM workspace.agents WHERE id = $1 FOR UPDATE`, lockedAgentID).Scan(
		&selectedAgentID,
	); err != nil {
		t.Fatalf("lock agent fixture: %v", err)
	}
	boundedRepository, err := NewRepository(pool, func() time.Time { return revokedAt })
	if err != nil {
		t.Fatalf("NewRepository(bounded) error = %v", err)
	}
	boundedRepository.operationMax = 100 * time.Millisecond
	if err := boundedRepository.RevokeAgent(
		context.Background(), foreignActor, lockedAgentID,
	); !errors.Is(err, ErrAgentAccessDenied) {
		t.Fatalf("RevokeAgent(foreign locked row) error = %v, want ErrAgentAccessDenied", err)
	}
	if err := boundedRepository.RevokeAgent(
		context.Background(), owner, lockedAgentID,
	); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("RevokeAgent(locked row) error = %v, want context.DeadlineExceeded", err)
	}
	assertAgentRevocation(t, ctx, pool, lockedAgentID, nil, 0)
	if err := lockingTx.Rollback(ctx); err != nil {
		t.Fatalf("release agent row lock: %v", err)
	}
	if err := boundedRepository.RevokeAgent(ctx, owner, lockedAgentID); err != nil {
		t.Fatalf("RevokeAgent(after lock release) error = %v", err)
	}
	assertAgentRevocation(t, ctx, pool, lockedAgentID, &revokedAt, 1)
}

func assertAgentRevocation(
	t *testing.T,
	ctx context.Context,
	reader workspaceAgentStateReader,
	agentID string,
	wantRevokedAt *time.Time,
	wantEventCount int,
) {
	t.Helper()
	assertAgentState(t, ctx, reader, agentID, nil, wantRevokedAt)
	var eventCount int
	if err := reader.QueryRow(ctx, `
		SELECT count(*) FROM outbox.events
		WHERE aggregate_type = 'agent' AND aggregate_id = $1`, agentID).Scan(&eventCount); err != nil {
		t.Fatalf("count agent revocation events: %v", err)
	}
	if eventCount != wantEventCount {
		t.Fatalf("agent revocation event count = %d, want %d", eventCount, wantEventCount)
	}
}
