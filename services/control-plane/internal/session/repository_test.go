package session

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRepositoryCreateRejectsInvalidBoundaries(t *testing.T) {
	repository := NewRepository()
	owner := newSessionTestActor(t, "0123456789abcdef0123456789abcdef")
	metadata, err := NewSession(
		owner,
		"2123456789abcdef0123456789abcdef",
		"3123456789abcdef0123456789abcdef",
		"4123456789abcdef0123456789abcdef",
		"5123456789abcdef0123456789abcdef",
		"eu-central-1",
		time.Date(2026, time.July, 19, 13, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}

	var nilRepository *Repository
	if err := nilRepository.Create(context.Background(), nil, metadata); !errors.Is(
		err, ErrSessionPersistenceUnavailable,
	) {
		t.Fatalf("nil Repository Create() error = %v, want ErrSessionPersistenceUnavailable", err)
	}
	if err := repository.Create(nil, nil, metadata); !errors.Is(err, ErrSessionPersistenceUnavailable) {
		t.Fatalf("Create(nil context) error = %v, want ErrSessionPersistenceUnavailable", err)
	}
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := repository.Create(canceledCtx, nil, metadata); !errors.Is(err, context.Canceled) {
		t.Fatalf("Create(canceled context) error = %v, want context.Canceled", err)
	}
	if err := repository.Create(context.Background(), nil, metadata); !errors.Is(
		err, ErrSessionPersistenceUnavailable,
	) {
		t.Fatalf("Create(nil transaction) error = %v, want ErrSessionPersistenceUnavailable", err)
	}
	repository.operationMax = 0
	if err := repository.Create(context.Background(), nil, metadata); !errors.Is(
		err, ErrSessionPersistenceUnavailable,
	) {
		t.Fatalf("Create(invalid operation maximum) error = %v, want ErrSessionPersistenceUnavailable", err)
	}
}

func TestRepositoryCreateOrGetRejectsInvalidBoundaries(t *testing.T) {
	repository := NewRepository()
	owner := newSessionTestActor(t, "0123456789abcdef0123456789abcdef")
	metadata, err := NewSession(
		owner,
		"2123456789abcdef0123456789abcdef",
		"3123456789abcdef0123456789abcdef",
		"4123456789abcdef0123456789abcdef",
		"5123456789abcdef0123456789abcdef",
		"eu-central-1",
		time.Date(2026, time.July, 19, 13, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}

	var nilRepository *Repository
	if _, err := nilRepository.CreateOrGet(context.Background(), nil, metadata); !errors.Is(
		err, ErrSessionPersistenceUnavailable,
	) {
		t.Fatalf("nil Repository CreateOrGet() error = %v, want ErrSessionPersistenceUnavailable", err)
	}
	if _, err := repository.CreateOrGet(nil, nil, metadata); !errors.Is(
		err, ErrSessionPersistenceUnavailable,
	) {
		t.Fatalf("CreateOrGet(nil context) error = %v, want ErrSessionPersistenceUnavailable", err)
	}
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := repository.CreateOrGet(canceledCtx, nil, metadata); !errors.Is(err, context.Canceled) {
		t.Fatalf("CreateOrGet(canceled context) error = %v, want context.Canceled", err)
	}
	if _, err := repository.CreateOrGet(context.Background(), nil, metadata); !errors.Is(
		err, ErrSessionPersistenceUnavailable,
	) {
		t.Fatalf("CreateOrGet(nil transaction) error = %v, want ErrSessionPersistenceUnavailable", err)
	}
	if _, err := repository.CreateOrGet(context.Background(), nil, Session{}); !errors.Is(
		err, ErrSessionPersistenceUnavailable,
	) {
		t.Fatalf("CreateOrGet(zero session, nil tx) error = %v, want persistence unavailable", err)
	}
	repository.operationMax = 0
	if _, err := repository.CreateOrGet(context.Background(), nil, metadata); !errors.Is(
		err, ErrSessionPersistenceUnavailable,
	) {
		t.Fatalf("CreateOrGet(invalid operation maximum) error = %v, want persistence unavailable", err)
	}
}
