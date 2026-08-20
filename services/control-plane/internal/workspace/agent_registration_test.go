package workspace

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/auth"
)

func TestRepositoryRegisterPairedAgentRejectsInvalidBoundaries(t *testing.T) {
	ownerID, err := auth.ParseUserID("0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatalf("ParseUserID() error = %v", err)
	}
	now := time.Date(2026, time.August, 3, 12, 0, 0, 0, time.UTC)
	repository, err := NewRepository(&workspaceTransactionStarterStub{}, func() time.Time { return now })
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}
	agentID := "1123456789abcdef0123456789abcdef"
	publicKey := newWorkspaceTestPublicKey(t, 0x42)

	var nilRepository *Repository
	if err := nilRepository.RegisterPairedAgent(
		context.Background(), nil, ownerID, agentID, "MacBook Agent", publicKey, "0.1.0", now,
	); !errors.Is(err, ErrWorkspacePersistenceUnavailable) {
		t.Fatalf("nil Repository RegisterPairedAgent() error = %v, want ErrWorkspacePersistenceUnavailable", err)
	}
	if err := repository.RegisterPairedAgent(
		nil, nil, ownerID, agentID, "MacBook Agent", publicKey, "0.1.0", now,
	); !errors.Is(err, ErrWorkspacePersistenceUnavailable) {
		t.Fatalf("RegisterPairedAgent(nil context) error = %v, want ErrWorkspacePersistenceUnavailable", err)
	}
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := repository.RegisterPairedAgent(
		canceledCtx, nil, ownerID, agentID, "MacBook Agent", publicKey, "0.1.0", now,
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("RegisterPairedAgent(canceled context) error = %v, want context.Canceled", err)
	}
	if err := repository.RegisterPairedAgent(
		context.Background(), nil, auth.UserID{}, agentID, "MacBook Agent", publicKey, "0.1.0", now,
	); !errors.Is(err, ErrAgentAccessDenied) {
		t.Fatalf("RegisterPairedAgent(zero owner) error = %v, want ErrAgentAccessDenied", err)
	}
	if err := repository.RegisterPairedAgent(
		context.Background(), nil, ownerID, "not-an-agent-id", "MacBook Agent", publicKey, "0.1.0", now,
	); !errors.Is(err, ErrInvalidAgent) {
		t.Fatalf("RegisterPairedAgent(invalid id) error = %v, want ErrInvalidAgent", err)
	}
	if err := repository.RegisterPairedAgent(
		context.Background(), nil, ownerID, agentID, "MacBook Agent", publicKey, "0.1.0", now.Add(time.Second),
	); !errors.Is(err, ErrInvalidAgent) {
		t.Fatalf("RegisterPairedAgent(future creation) error = %v, want ErrInvalidAgent", err)
	}
	zeroClockRepository, err := NewRepository(&workspaceTransactionStarterStub{}, func() time.Time { return time.Time{} })
	if err != nil {
		t.Fatalf("NewRepository(zero clock) error = %v", err)
	}
	if err := zeroClockRepository.RegisterPairedAgent(
		context.Background(), nil, ownerID, agentID, "MacBook Agent", publicKey, "0.1.0", now,
	); !errors.Is(err, ErrWorkspacePersistenceUnavailable) {
		t.Fatalf("RegisterPairedAgent(zero clock) error = %v, want ErrWorkspacePersistenceUnavailable", err)
	}
	if err := repository.RegisterPairedAgent(
		context.Background(), nil, ownerID, agentID, "MacBook Agent", publicKey, "0.1.0", now,
	); !errors.Is(err, ErrWorkspacePersistenceUnavailable) {
		t.Fatalf("RegisterPairedAgent(nil transaction) error = %v, want ErrWorkspacePersistenceUnavailable", err)
	}
}
