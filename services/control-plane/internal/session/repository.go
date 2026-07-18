package session

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const repositoryOperationTimeout = 5 * time.Second

var (
	ErrSessionAlreadyExists          = errors.New("session already exists")
	ErrSessionPersistenceUnavailable = errors.New("session persistence unavailable")
)

type Repository struct {
	operationMax time.Duration
}

func NewRepository() *Repository {
	return &Repository{operationMax: repositoryOperationTimeout}
}

func (repository *Repository) Create(ctx context.Context, tx pgx.Tx, session Session) error {
	if ctx == nil || repository == nil || repository.operationMax <= 0 {
		return ErrSessionPersistenceUnavailable
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if tx == nil {
		return ErrSessionPersistenceUnavailable
	}
	if session.id.String() == "" || session.ownerID.String() == "" || session.deviceID.String() == "" ||
		session.agentID.String() == "" || session.projectID.String() == "" ||
		!validRelayRegion(session.relayRegion) || session.startedAt.IsZero() {
		return ErrInvalidSession
	}
	operationCtx, cancelOperation := context.WithTimeout(ctx, repository.operationMax)
	defer cancelOperation()

	_, err := tx.Exec(operationCtx, `
		INSERT INTO session.sessions (
			id, user_id, device_id, agent_id, project_id, relay_region, started_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		session.id.String(), session.ownerID.String(), session.deviceID.String(), session.agentID.String(),
		session.projectID.String(), session.relayRegion, session.startedAt,
	)
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return context.DeadlineExceeded
	}
	var databaseErr *pgconn.PgError
	if errors.As(err, &databaseErr) && databaseErr.Code == "23505" &&
		databaseErr.ConstraintName == "sessions_pkey" {
		return ErrSessionAlreadyExists
	}
	return fmt.Errorf("%w: create: %w", ErrSessionPersistenceUnavailable, err)
}
