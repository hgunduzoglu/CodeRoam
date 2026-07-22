package session

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hgunduzoglu/coderoam/packages/go/ids"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/auth"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/device"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/workspace"
	"github.com/jackc/pgx/v5"
)

const (
	serviceOperationTimeout = 5 * time.Second
	serviceCleanupTimeout   = 5 * time.Second
)

type transactionStarter interface {
	Begin(context.Context) (pgx.Tx, error)
}

type deviceAuthorizer interface {
	Authorize(context.Context, pgx.Tx, auth.Actor, string) error
}

type workspaceAuthorizer interface {
	AuthorizeAgent(context.Context, pgx.Tx, auth.Actor, string) error
	AuthorizeProject(context.Context, pgx.Tx, auth.Actor, string, string) error
}

type sessionCreator interface {
	CreateOrGet(context.Context, pgx.Tx, Session) (Session, error)
}

var ErrSessionCommitOutcomeUnknown = errors.New("session commit outcome unknown")

type Service struct {
	transactions transactionStarter
	devices      deviceAuthorizer
	workspaces   workspaceAuthorizer
	sessions     sessionCreator
	relayRegion  string
	now          func() time.Time
	operationMax time.Duration
}

func NewService(
	transactions transactionStarter,
	devices deviceAuthorizer,
	workspaces workspaceAuthorizer,
	sessions sessionCreator,
	relayRegion string,
	now func() time.Time,
) (*Service, error) {
	if transactions == nil || devices == nil || workspaces == nil || sessions == nil {
		return nil, errors.New("session service repositories are required")
	}
	if !validRelayRegion(relayRegion) {
		return nil, errors.New("session service relay region is invalid")
	}
	if now == nil {
		return nil, errors.New("session service clock is required")
	}
	return &Service{
		transactions: transactions,
		devices:      devices,
		workspaces:   workspaces,
		sessions:     sessions,
		relayRegion:  relayRegion,
		now:          now,
		operationMax: serviceOperationTimeout,
	}, nil
}

func (service *Service) Start(
	ctx context.Context,
	actor auth.Actor,
	sessionID string,
	deviceID string,
	agentID string,
	projectID string,
) (started Session, err error) {
	if ctx == nil || service == nil || service.transactions == nil || service.devices == nil ||
		service.workspaces == nil || service.sessions == nil || service.now == nil ||
		service.operationMax <= 0 {
		return Session{}, ErrSessionPersistenceUnavailable
	}
	if err := ctx.Err(); err != nil {
		return Session{}, err
	}
	if _, ok := actor.UserID(); !ok {
		return Session{}, ErrSessionAccessDenied
	}
	if _, err := ids.Parse(sessionID); err != nil {
		return Session{}, ErrSessionAccessDenied
	}
	if _, err := ids.Parse(deviceID); err != nil {
		return Session{}, ErrSessionAccessDenied
	}
	if _, err := ids.Parse(agentID); err != nil {
		return Session{}, ErrSessionAccessDenied
	}
	if _, err := ids.Parse(projectID); err != nil {
		return Session{}, ErrSessionAccessDenied
	}

	startedAt := service.now().UTC()
	if startedAt.IsZero() {
		return Session{}, ErrSessionPersistenceUnavailable
	}
	metadata, err := NewSession(
		actor, sessionID, deviceID, agentID, projectID, service.relayRegion, startedAt,
	)
	if err != nil {
		return Session{}, ErrSessionPersistenceUnavailable
	}

	operationCtx, cancelOperation := context.WithTimeout(ctx, service.operationMax)
	defer cancelOperation()
	tx, beginErr := service.transactions.Begin(operationCtx)
	if tx == nil {
		return Session{}, sessionServiceError("begin", beginErr)
	}
	defer func() {
		rollbackCtx, cancelRollback := context.WithTimeout(context.WithoutCancel(ctx), serviceCleanupTimeout)
		defer cancelRollback()
		rollbackErr := tx.Rollback(rollbackCtx)
		if rollbackErr == nil || errors.Is(rollbackErr, pgx.ErrTxClosed) {
			return
		}
		rollbackErr = sessionServiceError("rollback", rollbackErr)
		if err == nil {
			err = rollbackErr
			started = Session{}
			return
		}
		err = errors.Join(err, rollbackErr)
	}()
	if beginErr != nil {
		return Session{}, sessionServiceError("begin", beginErr)
	}

	if err := service.devices.Authorize(operationCtx, tx, actor, deviceID); err != nil {
		return Session{}, sessionAuthorizationError("device", err)
	}
	if err := service.workspaces.AuthorizeAgent(operationCtx, tx, actor, agentID); err != nil {
		return Session{}, sessionAuthorizationError("agent", err)
	}
	if err := service.workspaces.AuthorizeProject(operationCtx, tx, actor, agentID, projectID); err != nil {
		return Session{}, sessionAuthorizationError("project", err)
	}
	persisted, err := service.sessions.CreateOrGet(operationCtx, tx, metadata)
	if err != nil {
		if errors.Is(err, ErrSessionAccessDenied) {
			return Session{}, ErrSessionAccessDenied
		}
		return Session{}, sessionServiceError("persist", err)
	}
	if err := tx.Commit(operationCtx); err != nil {
		return Session{}, sessionCommitError(err)
	}
	return persisted, nil
}

func sessionAuthorizationError(resource string, err error) error {
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return context.DeadlineExceeded
	}
	if errors.Is(err, device.ErrDeviceAccessDenied) || errors.Is(err, device.ErrInvalidDevice) ||
		errors.Is(err, workspace.ErrAgentAccessDenied) || errors.Is(err, workspace.ErrProjectAccessDenied) ||
		errors.Is(err, workspace.ErrInvalidAgent) || errors.Is(err, workspace.ErrInvalidProject) {
		return fmt.Errorf("%w: %s", ErrSessionAccessDenied, resource)
	}
	return sessionServiceError("authorize "+resource, err)
}

func sessionServiceError(operation string, err error) error {
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return context.DeadlineExceeded
	}
	if err == nil {
		return fmt.Errorf("%w: %s returned no transaction", ErrSessionPersistenceUnavailable, operation)
	}
	return fmt.Errorf("%w: %s: %w", ErrSessionPersistenceUnavailable, operation, err)
}

func sessionCommitError(err error) error {
	if err == nil {
		return ErrSessionCommitOutcomeUnknown
	}
	return fmt.Errorf("%w: %w", ErrSessionCommitOutcomeUnknown, err)
}
