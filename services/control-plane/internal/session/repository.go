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
	if !validSessionMetadata(session) {
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

func (repository *Repository) CreateOrGet(
	ctx context.Context,
	tx pgx.Tx,
	requested Session,
) (Session, error) {
	if ctx == nil || repository == nil || repository.operationMax <= 0 {
		return Session{}, ErrSessionPersistenceUnavailable
	}
	if err := ctx.Err(); err != nil {
		return Session{}, err
	}
	if tx == nil {
		return Session{}, ErrSessionPersistenceUnavailable
	}
	if !validSessionMetadata(requested) {
		return Session{}, ErrInvalidSession
	}
	operationCtx, cancelOperation := context.WithTimeout(ctx, repository.operationMax)
	defer cancelOperation()

	_, err := tx.Exec(operationCtx, `
		INSERT INTO session.sessions (
			id, user_id, device_id, agent_id, project_id, relay_region, started_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO NOTHING`,
		requested.id.String(), requested.ownerID.String(), requested.deviceID.String(),
		requested.agentID.String(), requested.projectID.String(), requested.relayRegion, requested.startedAt,
	)
	if err != nil {
		return Session{}, repositoryCreateError("create idempotent", err)
	}

	var deviceID, agentID, projectID, relayRegion string
	var startedAt time.Time
	var endedAt *time.Time
	var result *string
	err = tx.QueryRow(operationCtx, `
		SELECT device_id, agent_id, project_id, relay_region, started_at, ended_at, result
		FROM session.sessions
		WHERE id = $1 AND user_id = $2
		FOR SHARE`, requested.id.String(), requested.ownerID.String()).Scan(
		&deviceID, &agentID, &projectID, &relayRegion, &startedAt, &endedAt, &result,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, ErrSessionAccessDenied
	}
	if err != nil {
		return Session{}, repositoryCreateError("read idempotent", err)
	}
	stored, err := newSession(
		requested.ownerID, requested.id.String(), deviceID, agentID, projectID, relayRegion, startedAt,
	)
	if err != nil || endedAt != nil || result != nil ||
		stored.deviceID != requested.deviceID || stored.agentID != requested.agentID ||
		stored.projectID != requested.projectID || stored.relayRegion != requested.relayRegion {
		return Session{}, ErrSessionAccessDenied
	}
	return stored, nil
}

func validSessionMetadata(session Session) bool {
	return session.id.String() != "" && session.ownerID.String() != "" && session.deviceID.String() != "" &&
		session.agentID.String() != "" && session.projectID.String() != "" &&
		validRelayRegion(session.relayRegion) && !session.startedAt.IsZero()
}

func repositoryCreateError(operation string, err error) error {
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return context.DeadlineExceeded
	}
	return fmt.Errorf("%w: %s: %w", ErrSessionPersistenceUnavailable, operation, err)
}
