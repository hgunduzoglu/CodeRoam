package workspace

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/hgunduzoglu/coderoam/packages/go/postgresx"
	"github.com/jackc/pgx/v5"
)

func TestRepositoryListPairedAgentsIntegration(t *testing.T) {
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
		t.Fatalf("begin paired-agent-list transaction: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if err := tx.Rollback(cleanupCtx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			t.Errorf("rollback paired-agent-list transaction: %v", err)
		}
	})

	owner := newWorkspaceTestActor(t, "0123456789abcdef0123456789abcdef", "owner@example.com")
	foreign := newWorkspaceTestActor(t, "3123456789abcdef0123456789abcdef", "foreign@example.com")
	ownerID, _ := owner.UserID()
	foreignID, _ := foreign.UserID()
	now := time.Date(2026, time.August, 20, 12, 0, 0, 0, time.UTC)
	olderID := newWorkspaceIntegrationID(t)
	newerID := newWorkspaceIntegrationID(t)
	foreignIDValue := newWorkspaceIntegrationID(t)
	insertAgentFixture(t, ctx, tx, olderID, ownerID.String(), now.Add(-2*time.Hour), 0x61)
	insertAgentFixture(t, ctx, tx, newerID, ownerID.String(), now.Add(-time.Hour), 0x62)
	insertAgentFixture(t, ctx, tx, foreignIDValue, foreignID.String(), now.Add(-time.Hour), 0x63)
	lastSeenAt := now.Add(-30 * time.Minute)
	revokedAt := now.Add(-15 * time.Minute)
	if _, err := tx.Exec(
		ctx, `UPDATE workspace.agents SET last_seen_at = $1 WHERE id = $2`, lastSeenAt, newerID,
	); err != nil {
		t.Fatalf("set listed agent last-seen time: %v", err)
	}
	if _, err := tx.Exec(
		ctx, `UPDATE workspace.agents SET revoked_at = $1 WHERE id = $2`, revokedAt, olderID,
	); err != nil {
		t.Fatalf("revoke listed agent: %v", err)
	}
	repository, err := NewRepository(tx, func() time.Time { return now })
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}

	summaries, err := repository.ListPairedAgents(ctx, owner, 10)
	if err != nil {
		t.Fatalf("ListPairedAgents() error = %v", err)
	}
	if len(summaries) != 2 || summaries[0].ID != newerID || summaries[1].ID != olderID ||
		summaries[0].LastSeenAt == nil || !summaries[0].LastSeenAt.Equal(lastSeenAt) ||
		summaries[0].RevokedAt != nil || summaries[1].RevokedAt == nil ||
		!summaries[1].RevokedAt.Equal(revokedAt) || summaries[0].Fingerprint == "" {
		t.Fatalf("paired agent summaries = %+v", summaries)
	}
	limited, err := repository.ListPairedAgents(ctx, owner, 1)
	if err != nil || len(limited) != 1 || limited[0].ID != newerID {
		t.Fatalf("ListPairedAgents(limit) = (%+v, %v)", limited, err)
	}
	foreignSummaries, err := repository.ListPairedAgents(ctx, foreign, 10)
	if err != nil || len(foreignSummaries) != 1 || foreignSummaries[0].ID != foreignIDValue {
		t.Fatalf("ListPairedAgents(foreign) = (%+v, %v)", foreignSummaries, err)
	}

	if _, err := tx.Exec(
		ctx, `UPDATE workspace.agents SET created_at = $1 WHERE id = $2`, now.Add(time.Hour), newerID,
	); err != nil {
		t.Fatalf("corrupt listed agent creation time: %v", err)
	}
	if _, err := repository.ListPairedAgents(
		ctx, owner, 10,
	); !errors.Is(err, ErrWorkspacePersistenceUnavailable) {
		t.Fatalf("ListPairedAgents(future row) error = %v", err)
	}
}
