package workspace

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/hgunduzoglu/coderoam/packages/go/postgresx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestRepositoryAuthorizeProjectIntegration(t *testing.T) {
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
		t.Fatalf("begin project authorization transaction: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if err := tx.Rollback(cleanupCtx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			t.Errorf("rollback project authorization transaction: %v", err)
		}
	})

	owner := newWorkspaceTestActor(t, "0123456789abcdef0123456789abcdef", "owner@example.com")
	foreignActor := newWorkspaceTestActor(t, "3123456789abcdef0123456789abcdef", "foreign@example.com")
	ownerID, _ := owner.UserID()
	checkedAt := time.Date(2026, time.July, 19, 12, 0, 0, 0, time.UTC)
	agentCreatedAt := checkedAt.Add(-3 * time.Hour)
	activeAgentID := newWorkspaceIntegrationID(t)
	otherAgentID := newWorkspaceIntegrationID(t)
	insertAgentFixture(t, ctx, tx, activeAgentID, ownerID.String(), agentCreatedAt, 0x61)
	insertAgentFixture(t, ctx, tx, otherAgentID, ownerID.String(), agentCreatedAt, 0x62)
	environmentID, projectID := insertProjectHierarchyFixture(
		t, ctx, tx, ownerID.String(), activeAgentID, checkedAt.Add(-2*time.Hour), checkedAt.Add(-time.Hour),
	)
	repository, err := NewRepository(tx, func() time.Time { return checkedAt })
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}

	if err := repository.AuthorizeProject(ctx, tx, owner, activeAgentID, projectID); err != nil {
		t.Fatalf("AuthorizeProject(owner matching agent) error = %v", err)
	}
	assertProjectState(t, ctx, tx, projectID, environmentID, "/srv/coderoam/project")
	if err := repository.AuthorizeProject(
		ctx, tx, owner, otherAgentID, projectID,
	); !errors.Is(err, ErrProjectAccessDenied) {
		t.Fatalf("AuthorizeProject(wrong agent) error = %v, want ErrProjectAccessDenied", err)
	}
	if err := repository.AuthorizeProject(
		ctx, tx, foreignActor, activeAgentID, projectID,
	); !errors.Is(err, ErrProjectAccessDenied) {
		t.Fatalf("AuthorizeProject(foreign actor) error = %v, want ErrProjectAccessDenied", err)
	}
	if err := repository.AuthorizeProject(
		ctx, tx, owner, activeAgentID, newWorkspaceIntegrationID(t),
	); !errors.Is(err, ErrProjectAccessDenied) {
		t.Fatalf("AuthorizeProject(missing project) error = %v, want ErrProjectAccessDenied", err)
	}

	corruptEnvironmentID, corruptEnvironmentProjectID := insertProjectHierarchyFixture(
		t, ctx, tx, ownerID.String(), activeAgentID, checkedAt.Add(-2*time.Hour), checkedAt.Add(-time.Hour),
	)
	if _, err := tx.Exec(
		ctx, `UPDATE workspace.environments SET provider = $1 WHERE id = $2`,
		"linux\nforged", corruptEnvironmentID,
	); err != nil {
		t.Fatalf("corrupt environment provider: %v", err)
	}
	if err := repository.AuthorizeProject(
		ctx, tx, owner, activeAgentID, corruptEnvironmentProjectID,
	); !errors.Is(err, ErrProjectAccessDenied) {
		t.Fatalf("AuthorizeProject(corrupt environment) error = %v, want ErrProjectAccessDenied", err)
	}

	_, corruptProjectID := insertProjectHierarchyFixture(
		t, ctx, tx, ownerID.String(), activeAgentID, checkedAt.Add(-2*time.Hour), checkedAt.Add(-time.Hour),
	)
	if _, err := tx.Exec(
		ctx, `UPDATE workspace.projects SET root_path = $1 WHERE id = $2`,
		"/srv/coderoam/../escape", corruptProjectID,
	); err != nil {
		t.Fatalf("corrupt project root: %v", err)
	}
	if err := repository.AuthorizeProject(
		ctx, tx, owner, activeAgentID, corruptProjectID,
	); !errors.Is(err, ErrProjectAccessDenied) {
		t.Fatalf("AuthorizeProject(corrupt project) error = %v, want ErrProjectAccessDenied", err)
	}

	_, futureProjectID := insertProjectHierarchyFixture(
		t, ctx, tx, ownerID.String(), activeAgentID, checkedAt.Add(-time.Hour), checkedAt.Add(time.Hour),
	)
	if err := repository.AuthorizeProject(
		ctx, tx, owner, activeAgentID, futureProjectID,
	); !errors.Is(err, ErrProjectAccessDenied) {
		t.Fatalf("AuthorizeProject(future project) error = %v, want ErrProjectAccessDenied", err)
	}

	if _, err := tx.Exec(
		ctx, `UPDATE workspace.agents SET revoked_at = $1 WHERE id = $2`, checkedAt, activeAgentID,
	); err != nil {
		t.Fatalf("revoke linked agent fixture: %v", err)
	}
	if err := repository.AuthorizeProject(ctx, tx, owner, activeAgentID, projectID); err != nil {
		t.Fatalf("AuthorizeProject(revoked agent ownership) error = %v", err)
	}
	if err := repository.AuthorizeAgent(
		ctx, tx, owner, activeAgentID,
	); !errors.Is(err, ErrAgentAccessDenied) {
		t.Fatalf("AuthorizeAgent(revoked project agent) error = %v, want ErrAgentAccessDenied", err)
	}
}

func TestRepositoryAuthorizeProjectLockIntegration(t *testing.T) {
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
	checkedAt := time.Date(2026, time.July, 19, 12, 0, 0, 0, time.UTC)
	agentID := insertCommittedAgentFixture(t, ctx, pool, ownerID.String(), checkedAt.Add(-3*time.Hour), 0x63)
	environmentID, projectID := insertProjectHierarchyFixture(
		t, ctx, pool, ownerID.String(), agentID, checkedAt.Add(-2*time.Hour), checkedAt.Add(-time.Hour),
	)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if _, err := pool.Exec(cleanupCtx, `DELETE FROM workspace.projects WHERE id = $1`, projectID); err != nil {
			t.Errorf("delete committed project fixture: %v", err)
		}
		if _, err := pool.Exec(cleanupCtx, `DELETE FROM workspace.environments WHERE id = $1`, environmentID); err != nil {
			t.Errorf("delete committed environment fixture: %v", err)
		}
	})
	repository, err := NewRepository(pool, func() time.Time { return checkedAt })
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}
	repository.operationMax = 100 * time.Millisecond

	lockingTx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin project locking transaction: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		rollbackPossiblyClosedTransaction(t, cleanupCtx, lockingTx, "project locking")
	})
	if _, err := lockingTx.Exec(
		ctx, `SELECT id FROM workspace.projects WHERE id = $1 FOR UPDATE`, projectID,
	); err != nil {
		t.Fatalf("lock project fixture: %v", err)
	}
	boundedTx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin bounded project authorization transaction: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		rollbackPossiblyClosedTransaction(t, cleanupCtx, boundedTx, "bounded project authorization")
	})
	if err := repository.AuthorizeProject(
		context.Background(), boundedTx, owner, agentID, projectID,
	); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("AuthorizeProject(locked project) error = %v, want context.DeadlineExceeded", err)
	}
	rollbackPossiblyClosedTransaction(t, ctx, boundedTx, "bounded project authorization")
	if err := lockingTx.Rollback(ctx); err != nil {
		t.Fatalf("release project row lock: %v", err)
	}

	authorizationTx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin project authorization retry transaction: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		rollbackPossiblyClosedTransaction(t, cleanupCtx, authorizationTx, "project authorization retry")
	})
	if err := repository.AuthorizeProject(ctx, authorizationTx, owner, agentID, projectID); err != nil {
		t.Fatalf("AuthorizeProject(after lock release) error = %v", err)
	}
	assertUpdateTimesOut(t, pool, `UPDATE workspace.projects SET name = name WHERE id = $1`, projectID)
	assertUpdateTimesOut(t, pool, `UPDATE workspace.environments SET name = name WHERE id = $1`, environmentID)
	if err := authorizationTx.Commit(ctx); err != nil {
		t.Fatalf("commit project authorization transaction: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE workspace.projects SET name = name WHERE id = $1`, projectID); err != nil {
		t.Fatalf("update project after authorization commit: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE workspace.environments SET name = name WHERE id = $1`, environmentID); err != nil {
		t.Fatalf("update environment after authorization commit: %v", err)
	}
}

type workspaceFixtureWriter interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func insertProjectHierarchyFixture(
	t *testing.T,
	ctx context.Context,
	writer workspaceFixtureWriter,
	ownerID string,
	agentID string,
	environmentCreatedAt time.Time,
	projectCreatedAt time.Time,
) (string, string) {
	t.Helper()
	environmentID := newWorkspaceIntegrationID(t)
	projectID := newWorkspaceIntegrationID(t)
	if _, err := writer.Exec(ctx, `
		INSERT INTO workspace.environments (id, user_id, agent_id, name, provider, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		environmentID, ownerID, agentID, "Integration environment", "linux", environmentCreatedAt,
	); err != nil {
		t.Fatalf("insert environment fixture: %v", err)
	}
	if _, err := writer.Exec(ctx, `
		INSERT INTO workspace.projects (id, user_id, environment_id, name, root_path, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		projectID, ownerID, environmentID, "Integration project", "/srv/coderoam/project", projectCreatedAt,
	); err != nil {
		t.Fatalf("insert project fixture: %v", err)
	}
	return environmentID, projectID
}

func assertProjectState(
	t *testing.T,
	ctx context.Context,
	reader workspaceAgentStateReader,
	projectID string,
	wantEnvironmentID string,
	wantRootPath string,
) {
	t.Helper()
	var environmentID, rootPath string
	var lastOpenedAt *time.Time
	if err := reader.QueryRow(ctx, `
		SELECT environment_id, root_path, last_opened_at
		FROM workspace.projects WHERE id = $1`, projectID,
	).Scan(&environmentID, &rootPath, &lastOpenedAt); err != nil {
		t.Fatalf("read project state: %v", err)
	}
	if environmentID != wantEnvironmentID || rootPath != wantRootPath || lastOpenedAt != nil {
		t.Fatal("project authorization mutated or failed to preserve project state")
	}
}

func rollbackPossiblyClosedTransaction(t *testing.T, ctx context.Context, tx pgx.Tx, name string) {
	t.Helper()
	if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) && !tx.Conn().IsClosed() {
		t.Fatalf("rollback %s transaction: %v", name, err)
	}
}

func assertUpdateTimesOut(
	t *testing.T,
	database postgresx.TransactionStarter,
	query string,
	id string,
) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	tx, err := database.Begin(ctx)
	if err != nil {
		t.Fatalf("begin blocked update transaction: %v", err)
	}
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		rollbackPossiblyClosedTransaction(t, cleanupCtx, tx, "blocked update")
	}()
	_, updateErr := tx.Exec(ctx, query, id)
	if !errors.Is(updateErr, context.DeadlineExceeded) {
		t.Fatalf("blocked update error = %v, want context.DeadlineExceeded", updateErr)
	}
}
