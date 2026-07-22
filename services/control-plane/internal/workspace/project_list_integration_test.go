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

func TestRepositoryListProjectsIntegration(t *testing.T) {
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
		t.Fatalf("begin project-list transaction: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if err := tx.Rollback(cleanupCtx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			t.Errorf("rollback project-list transaction: %v", err)
		}
	})

	owner := newWorkspaceTestActor(t, "0123456789abcdef0123456789abcdef", "owner@example.com")
	foreign := newWorkspaceTestActor(t, "3123456789abcdef0123456789abcdef", "foreign@example.com")
	ownerID, _ := owner.UserID()
	foreignID, _ := foreign.UserID()
	now := time.Date(2026, time.July, 19, 23, 0, 0, 0, time.UTC)
	ownerAgentID := newWorkspaceIntegrationID(t)
	foreignAgentID := newWorkspaceIntegrationID(t)
	insertAgentFixture(t, ctx, tx, ownerAgentID, ownerID.String(), now.Add(-4*time.Hour), 0x71)
	insertAgentFixture(t, ctx, tx, foreignAgentID, foreignID.String(), now.Add(-4*time.Hour), 0x72)
	_, olderProjectID := insertProjectHierarchyFixture(
		t, ctx, tx, ownerID.String(), ownerAgentID, now.Add(-3*time.Hour), now.Add(-2*time.Hour),
	)
	newerEnvironmentID, newerProjectID := insertProjectHierarchyFixture(
		t, ctx, tx, ownerID.String(), ownerAgentID, now.Add(-2*time.Hour), now.Add(-time.Hour),
	)
	insertProjectHierarchyFixture(
		t, ctx, tx, foreignID.String(), foreignAgentID, now.Add(-2*time.Hour), now.Add(-time.Hour),
	)
	if _, err := tx.Exec(ctx, `UPDATE workspace.agents SET revoked_at = $1 WHERE id = $2`, now, ownerAgentID); err != nil {
		t.Fatalf("revoke listed agent: %v", err)
	}
	repository, err := NewRepository(tx, func() time.Time { return now })
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}

	summaries, err := repository.ListProjects(ctx, owner, 10)
	if err != nil {
		t.Fatalf("ListProjects() error = %v", err)
	}
	if len(summaries) != 2 || summaries[0].ID != newerProjectID || summaries[1].ID != olderProjectID ||
		summaries[0].AgentID != ownerAgentID || summaries[0].Name != "Integration project" ||
		summaries[0].EnvironmentName != "Integration environment" {
		t.Fatalf("project summaries = %+v", summaries)
	}
	limited, err := repository.ListProjects(ctx, owner, 1)
	if err != nil || len(limited) != 1 || limited[0].ID != newerProjectID {
		t.Fatalf("ListProjects(limit) = (%+v, %v)", limited, err)
	}
	foreignSummaries, err := repository.ListProjects(ctx, foreign, 10)
	if err != nil || len(foreignSummaries) != 1 {
		t.Fatalf("ListProjects(foreign) = (%+v, %v)", foreignSummaries, err)
	}

	if _, err := tx.Exec(ctx, `UPDATE workspace.projects SET root_path = '/srv/../escape' WHERE id = $1`, newerProjectID); err != nil {
		t.Fatalf("corrupt project root: %v", err)
	}
	if _, err := repository.ListProjects(ctx, owner, 10); !errors.Is(err, ErrWorkspacePersistenceUnavailable) {
		t.Fatalf("ListProjects(corrupt row) error = %v", err)
	}
	if _, err := tx.Exec(
		ctx, `UPDATE workspace.projects SET root_path = '/srv/coderoam/project' WHERE id = $1`, newerProjectID,
	); err != nil {
		t.Fatalf("restore project root: %v", err)
	}
	if _, err := tx.Exec(
		ctx, `UPDATE workspace.environments SET user_id = $1 WHERE id = $2`, foreignID.String(), newerEnvironmentID,
	); err != nil {
		t.Fatalf("corrupt environment owner: %v", err)
	}
	if _, err := repository.ListProjects(ctx, owner, 10); !errors.Is(err, ErrWorkspacePersistenceUnavailable) {
		t.Fatalf("ListProjects(cross-owner hierarchy) error = %v", err)
	}
	if _, err := tx.Exec(
		ctx, `UPDATE workspace.environments SET user_id = $1, agent_id = $2 WHERE id = $3`,
		ownerID.String(), newWorkspaceIntegrationID(t), newerEnvironmentID,
	); err != nil {
		t.Fatalf("corrupt environment agent: %v", err)
	}
	if _, err := repository.ListProjects(ctx, owner, 10); !errors.Is(err, ErrWorkspacePersistenceUnavailable) {
		t.Fatalf("ListProjects(orphaned agent) error = %v", err)
	}
}
