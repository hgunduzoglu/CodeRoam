package device

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/auth"
	"github.com/jackc/pgx/v5"
)

type transactionStarterStub struct {
	beginCalls int
	err        error
}

func (starter *transactionStarterStub) Begin(context.Context) (pgx.Tx, error) {
	starter.beginCalls++
	return nil, starter.err
}

func TestNewRepositoryRequiresDependencies(t *testing.T) {
	clock := func() time.Time { return time.Date(2026, time.July, 18, 15, 0, 0, 0, time.UTC) }
	if _, err := NewRepository(nil, clock); err == nil {
		t.Fatal("NewRepository(nil transactions) error = nil")
	}
	if _, err := NewRepository(&transactionStarterStub{}, nil); err == nil {
		t.Fatal("NewRepository(nil clock) error = nil")
	}
	repository, err := NewRepository(&transactionStarterStub{}, clock)
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}
	if repository.enqueue == nil {
		t.Fatal("NewRepository() did not install the outbox publisher")
	}
	if repository.operationMax != repositoryOperationTimeout {
		t.Fatalf("NewRepository() operation maximum = %v, want %v", repository.operationMax, repositoryOperationTimeout)
	}
}

func TestRepositoryRevokeRejectsInvalidBoundaries(t *testing.T) {
	owner := newTestActor(t, "0123456789abcdef0123456789abcdef", "owner@example.com")
	clock := func() time.Time { return time.Date(2026, time.July, 18, 15, 0, 0, 0, time.UTC) }
	starter := &transactionStarterStub{err: errors.New("database unavailable")}
	repository, err := NewRepository(starter, clock)
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}

	var nilRepository *Repository
	if err := nilRepository.Revoke(context.Background(), owner, "1123456789abcdef0123456789abcdef"); !errors.Is(err, ErrDevicePersistenceUnavailable) {
		t.Fatalf("nil Repository Revoke() error = %v, want ErrDevicePersistenceUnavailable", err)
	}
	if err := repository.Revoke(nil, owner, "1123456789abcdef0123456789abcdef"); !errors.Is(err, ErrDevicePersistenceUnavailable) {
		t.Fatalf("Revoke(nil context) error = %v, want ErrDevicePersistenceUnavailable", err)
	}
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := repository.Revoke(canceledCtx, owner, "1123456789abcdef0123456789abcdef"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Revoke(canceled context) error = %v, want context.Canceled", err)
	}
	if err := repository.Revoke(context.Background(), auth.Actor{}, "1123456789abcdef0123456789abcdef"); !errors.Is(err, ErrDeviceAccessDenied) {
		t.Fatalf("Revoke(zero actor) error = %v, want ErrDeviceAccessDenied", err)
	}
	if err := repository.Revoke(context.Background(), owner, "not-a-device-id"); !errors.Is(err, ErrInvalidDevice) {
		t.Fatalf("Revoke(invalid id) error = %v, want ErrInvalidDevice", err)
	}
	zeroClockRepository, err := NewRepository(starter, func() time.Time { return time.Time{} })
	if err != nil {
		t.Fatalf("NewRepository(zero clock) error = %v", err)
	}
	if err := zeroClockRepository.Revoke(context.Background(), owner, "1123456789abcdef0123456789abcdef"); !errors.Is(err, ErrDevicePersistenceUnavailable) {
		t.Fatalf("Revoke(zero clock) error = %v, want ErrDevicePersistenceUnavailable", err)
	}
	if starter.beginCalls != 0 {
		t.Fatalf("invalid boundaries began %d transactions, want 0", starter.beginCalls)
	}

	if err := repository.Revoke(context.Background(), owner, "1123456789abcdef0123456789abcdef"); !errors.Is(err, ErrDevicePersistenceUnavailable) {
		t.Fatalf("Revoke(begin failure) error = %v, want ErrDevicePersistenceUnavailable", err)
	}
	if starter.beginCalls != 1 {
		t.Fatalf("valid boundary began %d transactions, want 1", starter.beginCalls)
	}
}
