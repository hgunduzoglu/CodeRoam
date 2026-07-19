package outbox

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRepositoryClaimRejectsInvalidBoundaries(t *testing.T) {
	repository := NewRepository()
	now := time.Date(2026, time.July, 19, 18, 30, 0, 0, time.UTC)
	var nilRepository *Repository
	if _, _, err := nilRepository.Claim(context.Background(), nil, now); !errors.Is(
		err, ErrRepositoryUnavailable,
	) {
		t.Fatalf("nil Repository Claim() error = %v, want ErrRepositoryUnavailable", err)
	}
	if _, _, err := repository.Claim(nil, nil, now); !errors.Is(err, ErrRepositoryUnavailable) {
		t.Fatalf("Claim(nil context) error = %v, want ErrRepositoryUnavailable", err)
	}
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := repository.Claim(canceledCtx, nil, now); !errors.Is(err, context.Canceled) {
		t.Fatalf("Claim(canceled context) error = %v, want context.Canceled", err)
	}
	if _, _, err := repository.Claim(context.Background(), nil, now); !errors.Is(
		err, ErrRepositoryUnavailable,
	) {
		t.Fatalf("Claim(nil transaction) error = %v, want ErrRepositoryUnavailable", err)
	}
	if _, _, err := repository.Claim(context.Background(), nil, time.Time{}); !errors.Is(
		err, ErrRepositoryUnavailable,
	) {
		t.Fatalf("Claim(zero time) error = %v, want ErrRepositoryUnavailable", err)
	}
	repository.operationMax = 0
	if _, _, err := repository.Claim(context.Background(), nil, now); !errors.Is(
		err, ErrRepositoryUnavailable,
	) {
		t.Fatalf("Claim(invalid operation maximum) error = %v, want ErrRepositoryUnavailable", err)
	}
}

func TestRepositoryFinishRejectsInvalidBoundaries(t *testing.T) {
	repository := NewRepository()
	now := time.Date(2026, time.July, 19, 18, 30, 0, 0, time.UTC)
	claim := claimedEvent{locator: "(0,1)"}
	var nilRepository *Repository
	if err := nilRepository.Finish(context.Background(), nil, claim, finishCompleted, now); !errors.Is(
		err, ErrRepositoryUnavailable,
	) {
		t.Fatalf("nil Repository Finish() error = %v, want ErrRepositoryUnavailable", err)
	}
	if err := repository.Finish(nil, nil, claim, finishCompleted, now); !errors.Is(
		err, ErrRepositoryUnavailable,
	) {
		t.Fatalf("Finish(nil context) error = %v, want ErrRepositoryUnavailable", err)
	}
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := repository.Finish(canceledCtx, nil, claim, finishCompleted, now); !errors.Is(
		err, context.Canceled,
	) {
		t.Fatalf("Finish(canceled context) error = %v, want context.Canceled", err)
	}
	if err := repository.Finish(context.Background(), nil, claim, finishCompleted, now); !errors.Is(
		err, ErrRepositoryUnavailable,
	) {
		t.Fatalf("Finish(nil transaction) error = %v, want ErrRepositoryUnavailable", err)
	}
	if err := repository.Finish(context.Background(), nil, claimedEvent{}, finishCompleted, now); !errors.Is(
		err, ErrRepositoryUnavailable,
	) {
		t.Fatalf("Finish(empty claim) error = %v, want ErrRepositoryUnavailable", err)
	}
	if err := repository.Finish(context.Background(), nil, claim, finishCompleted, time.Time{}); !errors.Is(
		err, ErrRepositoryUnavailable,
	) {
		t.Fatalf("Finish(zero time) error = %v, want ErrRepositoryUnavailable", err)
	}
	repository.retryDelay = 0
	if err := repository.Finish(context.Background(), nil, claim, finishCompleted, now); !errors.Is(
		err, ErrRepositoryUnavailable,
	) {
		t.Fatalf("Finish(invalid retry delay) error = %v, want ErrRepositoryUnavailable", err)
	}
}
