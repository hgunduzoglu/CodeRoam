package workspace

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hgunduzoglu/coderoam/packages/go/cryptox"
	"github.com/hgunduzoglu/coderoam/packages/go/ids"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/auth"
	"github.com/jackc/pgx/v5"
)

const repositoryOperationTimeout = 5 * time.Second

var ErrWorkspacePersistenceUnavailable = errors.New("workspace persistence unavailable")

type Repository struct {
	now          func() time.Time
	operationMax time.Duration
}

func NewRepository(now func() time.Time) (*Repository, error) {
	if now == nil {
		return nil, errors.New("workspace repository clock is required")
	}
	return &Repository{
		now:          now,
		operationMax: repositoryOperationTimeout,
	}, nil
}

func (repository *Repository) AuthorizeAgent(
	ctx context.Context,
	tx pgx.Tx,
	actor auth.Actor,
	encodedAgentID string,
) error {
	if ctx == nil || repository == nil || repository.now == nil || repository.operationMax <= 0 {
		return ErrWorkspacePersistenceUnavailable
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	actorID, ok := actor.UserID()
	if !ok {
		return ErrAgentAccessDenied
	}
	agentID, err := ids.Parse(encodedAgentID)
	if err != nil {
		return fmt.Errorf("%w: id", ErrInvalidAgent)
	}
	checkedAt := repository.now().UTC()
	if checkedAt.IsZero() || tx == nil {
		return ErrWorkspacePersistenceUnavailable
	}
	operationCtx, cancelOperation := context.WithTimeout(ctx, repository.operationMax)
	defer cancelOperation()

	var name, version string
	var publicKeyBytes []byte
	var createdAt time.Time
	err = tx.QueryRow(operationCtx, `
		SELECT name, static_public_key, version, created_at
		FROM workspace.agents
		WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL
		  AND octet_length(name) BETWEEN 1 AND $3
		  AND octet_length(static_public_key) = 32
		  AND octet_length(version) BETWEEN 1 AND $4
		  AND created_at <= $5
		FOR SHARE`,
		agentID.String(), actorID.String(), maxAgentNameBytes, maxAgentVersionBytes, checkedAt).Scan(
		&name, &publicKeyBytes, &version, &createdAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrAgentAccessDenied
	}
	if err != nil {
		return workspacePersistenceError("authorize agent", err)
	}
	publicKey, err := cryptox.ParseX25519PublicKey(publicKeyBytes)
	if err != nil {
		return ErrAgentAccessDenied
	}
	storedAgent, err := NewAgent(actor, agentID.String(), name, publicKey, version, createdAt)
	if err != nil || !storedAgent.CanAuthorize(actor) {
		return ErrAgentAccessDenied
	}
	return nil
}

func workspacePersistenceError(operation string, err error) error {
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return context.DeadlineExceeded
	}
	return fmt.Errorf("%w: %s: %w", ErrWorkspacePersistenceUnavailable, operation, err)
}
