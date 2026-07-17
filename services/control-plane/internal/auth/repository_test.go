package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type recordingDatabase struct {
	called bool
}

func (database *recordingDatabase) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	database.called = true
	return pgconn.CommandTag{}, errors.New("unexpected query")
}

func (database *recordingDatabase) QueryRow(context.Context, string, ...any) pgx.Row {
	database.called = true
	return nil
}

func TestNewRepositoryRequiresDatabase(t *testing.T) {
	repository, err := NewRepository(nil)
	if err == nil || repository != nil {
		t.Fatalf("NewRepository(nil) = (%v, %v), want error", repository, err)
	}
}

func TestRepositoryRejectsZeroDomainValuesBeforeQuery(t *testing.T) {
	database := &recordingDatabase{}
	repository, err := NewRepository(database)
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}
	if err := repository.Create(context.Background(), User{}); !errors.Is(err, ErrInvalidUser) {
		t.Fatalf("Create() error = %v, want ErrInvalidUser", err)
	}
	if _, err := repository.FindByID(context.Background(), UserID{}); !errors.Is(err, ErrInvalidUser) {
		t.Fatalf("FindByID() error = %v, want ErrInvalidUser", err)
	}
	if database.called {
		t.Fatal("repository queried the database for an invalid domain value")
	}
}
