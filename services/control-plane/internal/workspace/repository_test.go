package workspace

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/auth"
)

func TestNewRepositoryRequiresClock(t *testing.T) {
	if _, err := NewRepository(nil); err == nil {
		t.Fatal("NewRepository(nil clock) error = nil")
	}
	repository, err := NewRepository(func() time.Time {
		return time.Date(2026, time.July, 18, 20, 0, 0, 0, time.UTC)
	})
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}
	if repository.operationMax != repositoryOperationTimeout {
		t.Fatalf("operationMax = %v, want %v", repository.operationMax, repositoryOperationTimeout)
	}
}

func TestRepositoryAuthorizeAgentRejectsInvalidBoundaries(t *testing.T) {
	owner := newWorkspaceTestActor(t, "0123456789abcdef0123456789abcdef", "owner@example.com")
	repository, err := NewRepository(func() time.Time {
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
	zeroClockRepository, err := NewRepository(func() time.Time { return time.Time{} })
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
