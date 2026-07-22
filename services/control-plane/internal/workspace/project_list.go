package workspace

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/hgunduzoglu/coderoam/packages/go/ids"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/auth"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const maxProjectListLimit = 100

var ErrInvalidProjectList = errors.New("invalid project list request")

type ProjectSummary struct {
	ID              string
	EnvironmentID   string
	AgentID         string
	Name            string
	EnvironmentName string
	CreatedAt       time.Time
}

func (summary ProjectSummary) Valid() bool {
	if _, err := ids.Parse(summary.ID); err != nil {
		return false
	}
	if _, err := ids.Parse(summary.EnvironmentID); err != nil {
		return false
	}
	if _, err := ids.Parse(summary.AgentID); err != nil {
		return false
	}
	validName := func(value string, maxBytes, maxRunes int) bool {
		return value != "" && len(value) <= maxBytes && utf8.ValidString(value) &&
			strings.TrimSpace(value) == value && !strings.ContainsFunc(value, unicode.IsControl) &&
			utf8.RuneCountInString(value) <= maxRunes
	}
	return validName(summary.Name, maxProjectNameBytes, maxProjectNameRunes) &&
		validName(summary.EnvironmentName, maxEnvironmentNameBytes, maxEnvironmentNameRunes) &&
		!summary.CreatedAt.IsZero()
}

func (repository *Repository) ListProjects(
	ctx context.Context,
	actor auth.Actor,
	limit int,
) (summaries []ProjectSummary, err error) {
	if ctx == nil || repository == nil || repository.transactions == nil || repository.now == nil ||
		repository.operationMax <= 0 {
		return nil, ErrWorkspacePersistenceUnavailable
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ownerID, ok := actor.UserID()
	if !ok {
		return nil, ErrProjectAccessDenied
	}
	if limit < 1 || limit > maxProjectListLimit {
		return nil, ErrInvalidProjectList
	}
	checkedAt := repository.now().UTC()
	if checkedAt.IsZero() {
		return nil, ErrWorkspacePersistenceUnavailable
	}
	operationCtx, cancelOperation := context.WithTimeout(ctx, repository.operationMax)
	defer cancelOperation()
	tx, err := repository.transactions.Begin(operationCtx)
	if err != nil {
		return nil, workspacePersistenceError("begin project list", err)
	}
	if tx == nil {
		return nil, ErrWorkspacePersistenceUnavailable
	}
	defer func() {
		rollbackCtx, cancelRollback := context.WithTimeout(context.WithoutCancel(ctx), transactionCleanupTimeout)
		defer cancelRollback()
		rollbackErr := tx.Rollback(rollbackCtx)
		if rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) && err == nil {
			summaries = nil
			err = workspacePersistenceError("rollback project list", rollbackErr)
		}
	}()

	rows, err := tx.Query(operationCtx, `
		SELECT CASE WHEN octet_length(p.id) = 32 THEN p.id END,
		       CASE WHEN octet_length(p.name) BETWEEN 1 AND $2 THEN p.name END,
		       CASE WHEN octet_length(p.root_path) BETWEEN 1 AND $3 THEN p.root_path END,
		       p.created_at,
		       CASE WHEN octet_length(e.id) = 32 THEN e.id END,
		       CASE WHEN octet_length(e.name) BETWEEN 1 AND $4 THEN e.name END,
		       CASE WHEN octet_length(e.provider) BETWEEN 1 AND $5 THEN e.provider END,
		       e.created_at,
		       CASE WHEN octet_length(e.agent_id) = 32 THEN e.agent_id END,
		       a.created_at
		FROM workspace.projects AS p
		LEFT JOIN workspace.environments AS e
		  ON e.id = p.environment_id AND e.user_id = p.user_id
		LEFT JOIN workspace.agents AS a
		  ON a.id = e.agent_id AND a.user_id = e.user_id
		WHERE p.user_id = $1
		ORDER BY p.created_at DESC, p.id
		LIMIT $6`,
		ownerID.String(), maxProjectNameBytes, maxProjectRootBytes, maxEnvironmentNameBytes,
		maxEnvironmentProviderBytes, limit,
	)
	if err != nil {
		return nil, workspacePersistenceError("query project list", err)
	}
	defer rows.Close()
	for rows.Next() {
		var projectID, projectName, rootPath, environmentID, environmentName, provider, agentID *string
		var projectCreatedAt, environmentCreatedAt, agentCreatedAt pgtype.Timestamptz
		if err := rows.Scan(
			&projectID, &projectName, &rootPath, &projectCreatedAt,
			&environmentID, &environmentName, &provider, &environmentCreatedAt,
			&agentID, &agentCreatedAt,
		); err != nil {
			return nil, workspacePersistenceError("scan project list", err)
		}
		if projectID == nil || projectName == nil || rootPath == nil || environmentID == nil ||
			environmentName == nil || provider == nil || agentID == nil ||
			!finiteTimestamp(projectCreatedAt) || !finiteTimestamp(environmentCreatedAt) ||
			!finiteTimestamp(agentCreatedAt) {
			return nil, fmt.Errorf("%w: corrupt project list row", ErrWorkspacePersistenceUnavailable)
		}
		parsedAgentID, err := ids.Parse(*agentID)
		if err != nil {
			return nil, fmt.Errorf("%w: corrupt project list row", ErrWorkspacePersistenceUnavailable)
		}
		environment, err := newEnvironment(
			ownerID, *environmentID, parsedAgentID, *environmentName, *provider,
			agentCreatedAt.Time, environmentCreatedAt.Time,
		)
		if err != nil {
			return nil, fmt.Errorf("%w: corrupt project list row", ErrWorkspacePersistenceUnavailable)
		}
		project, err := NewProject(
			actor, *projectID, environment, *projectName, *rootPath, projectCreatedAt.Time,
		)
		if err != nil {
			return nil, fmt.Errorf("%w: corrupt project list row", ErrWorkspacePersistenceUnavailable)
		}
		if project.createdAt.After(checkedAt) {
			return nil, fmt.Errorf("%w: future project list row", ErrWorkspacePersistenceUnavailable)
		}
		summaries = append(summaries, ProjectSummary{
			ID: project.id.String(), EnvironmentID: environment.id.String(), AgentID: environment.agentID.String(),
			Name: project.name, EnvironmentName: environment.name, CreatedAt: project.createdAt,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, workspacePersistenceError("iterate project list", err)
	}
	if err := tx.Commit(operationCtx); err != nil {
		return nil, workspacePersistenceError("commit project list", err)
	}
	return summaries, nil
}

func finiteTimestamp(value pgtype.Timestamptz) bool {
	return value.Valid && value.InfinityModifier == pgtype.Finite && !value.Time.IsZero()
}
