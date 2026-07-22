package workspace

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/auth"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/outbox"
	"github.com/jackc/pgx/v5"
)

type workspaceTransactionStarterStub struct {
	beginCalls int
	err        error
}

func (starter *workspaceTransactionStarterStub) Begin(context.Context) (pgx.Tx, error) {
	starter.beginCalls++
	return nil, starter.err
}

func TestNewRepositoryRequiresDependencies(t *testing.T) {
	clock := func() time.Time { return time.Date(2026, time.July, 18, 20, 0, 0, 0, time.UTC) }
	if _, err := NewRepository(nil, clock); err == nil {
		t.Fatal("NewRepository(nil transactions) error = nil")
	}
	if _, err := NewRepository(&workspaceTransactionStarterStub{}, nil); err == nil {
		t.Fatal("NewRepository(nil clock) error = nil")
	}
	repository, err := NewRepository(&workspaceTransactionStarterStub{}, clock)
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}
	if repository.enqueue == nil {
		t.Fatal("NewRepository() did not install the outbox publisher")
	}
	if repository.operationMax != repositoryOperationTimeout {
		t.Fatalf("operationMax = %v, want %v", repository.operationMax, repositoryOperationTimeout)
	}
}

func TestRepositoryAuthorizeAgentRejectsInvalidBoundaries(t *testing.T) {
	owner := newWorkspaceTestActor(t, "0123456789abcdef0123456789abcdef", "owner@example.com")
	starter := &workspaceTransactionStarterStub{}
	repository, err := NewRepository(starter, func() time.Time {
		return time.Date(2026, time.July, 18, 20, 0, 0, 0, time.UTC)
	})
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}

	var nilRepository *Repository
	if err := nilRepository.AuthorizeAgent(
		context.Background(), nil, owner, "1123456789abcdef0123456789abcdef",
	); !errors.Is(err, ErrWorkspacePersistenceUnavailable) {
		t.Fatalf("nil Repository AuthorizeAgent() error = %v, want ErrWorkspacePersistenceUnavailable", err)
	}
	if err := repository.AuthorizeAgent(
		nil, nil, owner, "1123456789abcdef0123456789abcdef",
	); !errors.Is(err, ErrWorkspacePersistenceUnavailable) {
		t.Fatalf("AuthorizeAgent(nil context) error = %v, want ErrWorkspacePersistenceUnavailable", err)
	}
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := repository.AuthorizeAgent(
		canceledCtx, nil, owner, "1123456789abcdef0123456789abcdef",
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("AuthorizeAgent(canceled context) error = %v, want context.Canceled", err)
	}
	if err := repository.AuthorizeAgent(
		context.Background(), nil, auth.Actor{}, "1123456789abcdef0123456789abcdef",
	); !errors.Is(err, ErrAgentAccessDenied) {
		t.Fatalf("AuthorizeAgent(zero actor) error = %v, want ErrAgentAccessDenied", err)
	}
	if err := repository.AuthorizeAgent(
		context.Background(), nil, owner, "not-an-agent-id",
	); !errors.Is(err, ErrInvalidAgent) {
		t.Fatalf("AuthorizeAgent(invalid id) error = %v, want ErrInvalidAgent", err)
	}
	zeroClockRepository, err := NewRepository(starter, func() time.Time { return time.Time{} })
	if err != nil {
		t.Fatalf("NewRepository(zero clock) error = %v", err)
	}
	if err := zeroClockRepository.AuthorizeAgent(
		context.Background(), nil, owner, "1123456789abcdef0123456789abcdef",
	); !errors.Is(err, ErrWorkspacePersistenceUnavailable) {
		t.Fatalf("AuthorizeAgent(zero clock) error = %v, want ErrWorkspacePersistenceUnavailable", err)
	}
	if err := repository.AuthorizeAgent(
		context.Background(), nil, owner, "1123456789abcdef0123456789abcdef",
	); !errors.Is(err, ErrWorkspacePersistenceUnavailable) {
		t.Fatalf("AuthorizeAgent(nil transaction) error = %v, want ErrWorkspacePersistenceUnavailable", err)
	}
	repository.operationMax = 0
	if err := repository.AuthorizeAgent(
		context.Background(), nil, owner, "1123456789abcdef0123456789abcdef",
	); !errors.Is(err, ErrWorkspacePersistenceUnavailable) {
		t.Fatalf("AuthorizeAgent(invalid operation maximum) error = %v, want ErrWorkspacePersistenceUnavailable", err)
	}
}

func TestRepositoryAuthorizeProjectRejectsInvalidBoundaries(t *testing.T) {
	owner := newWorkspaceTestActor(t, "0123456789abcdef0123456789abcdef", "owner@example.com")
	starter := &workspaceTransactionStarterStub{}
	repository, err := NewRepository(starter, func() time.Time {
		return time.Date(2026, time.July, 19, 12, 0, 0, 0, time.UTC)
	})
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}
	agentID := "1123456789abcdef0123456789abcdef"
	projectID := "2123456789abcdef0123456789abcdef"

	var nilRepository *Repository
	if err := nilRepository.AuthorizeProject(
		context.Background(), nil, owner, agentID, projectID,
	); !errors.Is(err, ErrWorkspacePersistenceUnavailable) {
		t.Fatalf("nil Repository AuthorizeProject() error = %v, want ErrWorkspacePersistenceUnavailable", err)
	}
	if err := repository.AuthorizeProject(
		nil, nil, owner, agentID, projectID,
	); !errors.Is(err, ErrWorkspacePersistenceUnavailable) {
		t.Fatalf("AuthorizeProject(nil context) error = %v, want ErrWorkspacePersistenceUnavailable", err)
	}
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := repository.AuthorizeProject(
		canceledCtx, nil, owner, agentID, projectID,
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("AuthorizeProject(canceled context) error = %v, want context.Canceled", err)
	}
	if err := repository.AuthorizeProject(
		context.Background(), nil, auth.Actor{}, agentID, projectID,
	); !errors.Is(err, ErrProjectAccessDenied) {
		t.Fatalf("AuthorizeProject(zero actor) error = %v, want ErrProjectAccessDenied", err)
	}
	if err := repository.AuthorizeProject(
		context.Background(), nil, owner, "not-an-agent-id", projectID,
	); !errors.Is(err, ErrInvalidAgent) {
		t.Fatalf("AuthorizeProject(invalid agent id) error = %v, want ErrInvalidAgent", err)
	}
	if err := repository.AuthorizeProject(
		context.Background(), nil, owner, agentID, "not-a-project-id",
	); !errors.Is(err, ErrInvalidProject) {
		t.Fatalf("AuthorizeProject(invalid project id) error = %v, want ErrInvalidProject", err)
	}
	zeroClockRepository, err := NewRepository(starter, func() time.Time { return time.Time{} })
	if err != nil {
		t.Fatalf("NewRepository(zero clock) error = %v", err)
	}
	if err := zeroClockRepository.AuthorizeProject(
		context.Background(), nil, owner, agentID, projectID,
	); !errors.Is(err, ErrWorkspacePersistenceUnavailable) {
		t.Fatalf("AuthorizeProject(zero clock) error = %v, want ErrWorkspacePersistenceUnavailable", err)
	}
	if err := repository.AuthorizeProject(
		context.Background(), nil, owner, agentID, projectID,
	); !errors.Is(err, ErrWorkspacePersistenceUnavailable) {
		t.Fatalf("AuthorizeProject(nil transaction) error = %v, want ErrWorkspacePersistenceUnavailable", err)
	}
	repository.operationMax = 0
	if err := repository.AuthorizeProject(
		context.Background(), nil, owner, agentID, projectID,
	); !errors.Is(err, ErrWorkspacePersistenceUnavailable) {
		t.Fatalf("AuthorizeProject(invalid operation maximum) error = %v, want ErrWorkspacePersistenceUnavailable", err)
	}
}

func TestRepositoryRevokeAgentRejectsInvalidBoundaries(t *testing.T) {
	owner := newWorkspaceTestActor(t, "0123456789abcdef0123456789abcdef", "owner@example.com")
	clock := func() time.Time { return time.Date(2026, time.July, 19, 9, 0, 0, 0, time.UTC) }
	starter := &workspaceTransactionStarterStub{err: errors.New("database unavailable")}
	repository, err := NewRepository(starter, clock)
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}

	var nilRepository *Repository
	if err := nilRepository.RevokeAgent(
		context.Background(), owner, "1123456789abcdef0123456789abcdef",
	); !errors.Is(err, ErrWorkspacePersistenceUnavailable) {
		t.Fatalf("nil Repository RevokeAgent() error = %v, want ErrWorkspacePersistenceUnavailable", err)
	}
	if err := repository.RevokeAgent(
		nil, owner, "1123456789abcdef0123456789abcdef",
	); !errors.Is(err, ErrWorkspacePersistenceUnavailable) {
		t.Fatalf("RevokeAgent(nil context) error = %v, want ErrWorkspacePersistenceUnavailable", err)
	}
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := repository.RevokeAgent(
		canceledCtx, owner, "1123456789abcdef0123456789abcdef",
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("RevokeAgent(canceled context) error = %v, want context.Canceled", err)
	}
	if err := repository.RevokeAgent(
		context.Background(), auth.Actor{}, "1123456789abcdef0123456789abcdef",
	); !errors.Is(err, ErrAgentAccessDenied) {
		t.Fatalf("RevokeAgent(zero actor) error = %v, want ErrAgentAccessDenied", err)
	}
	if err := repository.RevokeAgent(
		context.Background(), owner, "not-an-agent-id",
	); !errors.Is(err, ErrInvalidAgent) {
		t.Fatalf("RevokeAgent(invalid id) error = %v, want ErrInvalidAgent", err)
	}
	zeroClockRepository, err := NewRepository(starter, func() time.Time { return time.Time{} })
	if err != nil {
		t.Fatalf("NewRepository(zero clock) error = %v", err)
	}
	if err := zeroClockRepository.RevokeAgent(
		context.Background(), owner, "1123456789abcdef0123456789abcdef",
	); !errors.Is(err, ErrWorkspacePersistenceUnavailable) {
		t.Fatalf("RevokeAgent(zero clock) error = %v, want ErrWorkspacePersistenceUnavailable", err)
	}
	repository.enqueue = nil
	if err := repository.RevokeAgent(
		context.Background(), owner, "1123456789abcdef0123456789abcdef",
	); !errors.Is(err, ErrWorkspacePersistenceUnavailable) {
		t.Fatalf("RevokeAgent(nil enqueue) error = %v, want ErrWorkspacePersistenceUnavailable", err)
	}
	repository.enqueue = func(context.Context, pgx.Tx, outbox.Event) error { return nil }
	repository.operationMax = 0
	if err := repository.RevokeAgent(
		context.Background(), owner, "1123456789abcdef0123456789abcdef",
	); !errors.Is(err, ErrWorkspacePersistenceUnavailable) {
		t.Fatalf("RevokeAgent(invalid operation maximum) error = %v, want ErrWorkspacePersistenceUnavailable", err)
	}
	if starter.beginCalls != 0 {
		t.Fatalf("invalid boundaries began %d transactions, want 0", starter.beginCalls)
	}

	repository.operationMax = repositoryOperationTimeout
	if err := repository.RevokeAgent(
		context.Background(), owner, "1123456789abcdef0123456789abcdef",
	); !errors.Is(err, ErrWorkspacePersistenceUnavailable) {
		t.Fatalf("RevokeAgent(begin failure) error = %v, want ErrWorkspacePersistenceUnavailable", err)
	}
	if starter.beginCalls != 1 {
		t.Fatalf("valid boundary began %d transactions, want 1", starter.beginCalls)
	}
}
