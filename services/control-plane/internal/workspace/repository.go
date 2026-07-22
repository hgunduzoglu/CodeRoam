package workspace

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hgunduzoglu/coderoam/packages/go/cryptox"
	"github.com/hgunduzoglu/coderoam/packages/go/ids"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/auth"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/outbox"
	"github.com/jackc/pgx/v5"
)

const (
	repositoryOperationTimeout = 5 * time.Second
	transactionCleanupTimeout  = 5 * time.Second
)

var ErrWorkspacePersistenceUnavailable = errors.New("workspace persistence unavailable")

type transactionStarter interface {
	Begin(context.Context) (pgx.Tx, error)
}

type enqueueEvent func(context.Context, pgx.Tx, outbox.Event) error

type Repository struct {
	transactions transactionStarter
	now          func() time.Time
	enqueue      enqueueEvent
	operationMax time.Duration
}

func NewRepository(transactions transactionStarter, now func() time.Time) (*Repository, error) {
	if transactions == nil {
		return nil, errors.New("workspace repository transaction starter is required")
	}
	if now == nil {
		return nil, errors.New("workspace repository clock is required")
	}
	return &Repository{
		transactions: transactions,
		now:          now,
		enqueue:      outbox.Enqueue,
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

func (repository *Repository) AuthorizeProject(
	ctx context.Context,
	tx pgx.Tx,
	actor auth.Actor,
	encodedAgentID string,
	encodedProjectID string,
) error {
	if ctx == nil || repository == nil || repository.now == nil || repository.operationMax <= 0 {
		return ErrWorkspacePersistenceUnavailable
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	actorID, ok := actor.UserID()
	if !ok {
		return ErrProjectAccessDenied
	}
	agentID, err := ids.Parse(encodedAgentID)
	if err != nil {
		return fmt.Errorf("%w: id", ErrInvalidAgent)
	}
	projectID, err := ids.Parse(encodedProjectID)
	if err != nil {
		return fmt.Errorf("%w: id", ErrInvalidProject)
	}
	checkedAt := repository.now().UTC()
	if checkedAt.IsZero() || tx == nil {
		return ErrWorkspacePersistenceUnavailable
	}
	operationCtx, cancelOperation := context.WithTimeout(ctx, repository.operationMax)
	defer cancelOperation()

	var projectName, rootPath, environmentID, environmentName, provider string
	var projectCreatedAt, environmentCreatedAt, agentCreatedAt time.Time
	err = tx.QueryRow(operationCtx, `
		SELECT p.name, p.root_path, p.created_at,
		       e.id, e.name, e.provider, e.created_at,
		       a.created_at
		FROM workspace.projects AS p
		JOIN workspace.environments AS e
		  ON e.id = p.environment_id AND e.user_id = p.user_id
		JOIN workspace.agents AS a
		  ON a.id = e.agent_id AND a.user_id = e.user_id
		WHERE p.id = $1 AND p.user_id = $2 AND a.id = $3
		  AND octet_length(p.name) BETWEEN 1 AND $4
		  AND octet_length(p.root_path) BETWEEN 1 AND $5
		  AND octet_length(e.name) BETWEEN 1 AND $6
		  AND octet_length(e.provider) BETWEEN 1 AND $7
		  AND a.created_at <= e.created_at
		  AND e.created_at <= p.created_at
		  AND p.created_at <= $8
		FOR SHARE OF p, e`,
		projectID.String(), actorID.String(), agentID.String(), maxProjectNameBytes,
		maxProjectRootBytes, maxEnvironmentNameBytes, maxEnvironmentProviderBytes, checkedAt,
	).Scan(
		&projectName, &rootPath, &projectCreatedAt,
		&environmentID, &environmentName, &provider, &environmentCreatedAt,
		&agentCreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrProjectAccessDenied
	}
	if err != nil {
		return workspacePersistenceError("authorize project", err)
	}
	environment, err := newEnvironment(
		actorID, environmentID, agentID, environmentName, provider, agentCreatedAt, environmentCreatedAt,
	)
	if err != nil {
		return ErrProjectAccessDenied
	}
	project, err := NewProject(
		actor, projectID.String(), environment, projectName, rootPath, projectCreatedAt,
	)
	if err != nil || !project.OwnedBy(actor) {
		return ErrProjectAccessDenied
	}
	return nil
}

func (repository *Repository) RevokeAgent(
	ctx context.Context,
	actor auth.Actor,
	encodedAgentID string,
) (err error) {
	if ctx == nil || repository == nil || repository.transactions == nil || repository.now == nil ||
		repository.enqueue == nil || repository.operationMax <= 0 {
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
	revokedAt := repository.now().UTC()
	if revokedAt.IsZero() {
		return ErrWorkspacePersistenceUnavailable
	}
	operationCtx, cancelOperation := context.WithTimeout(ctx, repository.operationMax)
	defer cancelOperation()

	tx, err := repository.transactions.Begin(operationCtx)
	if err != nil {
		return workspacePersistenceError("begin agent revocation", err)
	}
	if tx == nil {
		return ErrWorkspacePersistenceUnavailable
	}
	defer func() {
		rollbackCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), transactionCleanupTimeout)
		defer cancel()
		rollbackErr := tx.Rollback(rollbackCtx)
		if rollbackErr == nil || errors.Is(rollbackErr, pgx.ErrTxClosed) {
			return
		}
		rollbackErr = workspacePersistenceError("rollback agent revocation", rollbackErr)
		if err == nil {
			err = rollbackErr
			return
		}
		err = errors.Join(err, rollbackErr)
	}()

	var createdAt time.Time
	var storedRevokedAt *time.Time
	err = tx.QueryRow(operationCtx, `
		SELECT created_at, revoked_at
		FROM workspace.agents
		WHERE id = $1 AND user_id = $2
		FOR UPDATE`, agentID.String(), actorID.String()).Scan(&createdAt, &storedRevokedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrAgentAccessDenied
	}
	if err != nil {
		return workspacePersistenceError("lock agent for revocation", err)
	}
	if storedRevokedAt != nil {
		if err := tx.Commit(operationCtx); err != nil {
			return workspacePersistenceError("finish idempotent agent revocation", err)
		}
		return nil
	}
	if createdAt.IsZero() || revokedAt.Before(createdAt) {
		return fmt.Errorf("%w: stored creation time", ErrInvalidAgent)
	}

	event, err := outbox.NewEvent(outbox.EventAgentRevoked, agentID, revokedAt)
	if err != nil {
		return workspacePersistenceError("create agent revocation event", err)
	}
	result, err := tx.Exec(operationCtx, `
		UPDATE workspace.agents
		SET revoked_at = $1
		WHERE id = $2 AND user_id = $3 AND revoked_at IS NULL`,
		revokedAt, agentID.String(), actorID.String())
	if err != nil {
		return workspacePersistenceError("persist agent revocation", err)
	}
	if result.RowsAffected() != 1 {
		return fmt.Errorf(
			"%w: agent revocation update affected %d rows",
			ErrWorkspacePersistenceUnavailable,
			result.RowsAffected(),
		)
	}
	if err := repository.enqueue(operationCtx, tx, event); err != nil {
		return workspacePersistenceError("enqueue agent revocation", err)
	}
	if err := tx.Commit(operationCtx); err != nil {
		return workspacePersistenceError("commit agent revocation", err)
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
