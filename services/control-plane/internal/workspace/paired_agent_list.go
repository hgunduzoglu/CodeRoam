package workspace

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/hgunduzoglu/coderoam/packages/go/cryptox"
	"github.com/hgunduzoglu/coderoam/packages/go/ids"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/auth"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	maxPairedAgentListLimit          = 100
	canonicalX25519FingerprintLength = len("x25519-sha256:") + 64
)

var ErrInvalidPairedAgentList = errors.New("invalid paired agent list request")

// PairedAgentSummary contains owner-visible trust metadata and no key material.
// Revoked agents remain visible so management surfaces can explain their state.
type PairedAgentSummary struct {
	ID          string
	Name        string
	Version     string
	Fingerprint string
	CreatedAt   time.Time
	LastSeenAt  *time.Time
	RevokedAt   *time.Time
}

func (summary PairedAgentSummary) Valid() bool {
	if _, err := ids.Parse(summary.ID); err != nil {
		return false
	}
	if summary.Name == "" || len(summary.Name) > maxAgentNameBytes ||
		!utf8.ValidString(summary.Name) || strings.TrimSpace(summary.Name) != summary.Name ||
		strings.ContainsFunc(summary.Name, unicode.IsControl) ||
		utf8.RuneCountInString(summary.Name) > maxAgentNameRunes {
		return false
	}
	if summary.Version == "" || len(summary.Version) > maxAgentVersionBytes ||
		!utf8.ValidString(summary.Version) || strings.TrimSpace(summary.Version) != summary.Version ||
		strings.ContainsFunc(summary.Version, unicode.IsControl) {
		return false
	}
	if _, err := cryptox.ParseX25519Fingerprint(summary.Fingerprint); err != nil ||
		summary.CreatedAt.IsZero() {
		return false
	}
	if summary.LastSeenAt != nil &&
		(summary.LastSeenAt.IsZero() || summary.LastSeenAt.Before(summary.CreatedAt)) {
		return false
	}
	return summary.RevokedAt == nil ||
		(!summary.RevokedAt.IsZero() && !summary.RevokedAt.Before(summary.CreatedAt))
}

// ListPairedAgents returns a bounded owner-scoped management view. Callers
// must still use AuthorizeAgent for trust decisions.
func (repository *Repository) ListPairedAgents(
	ctx context.Context,
	actor auth.Actor,
	limit int,
) (summaries []PairedAgentSummary, err error) {
	if ctx == nil || repository == nil || repository.transactions == nil || repository.now == nil ||
		repository.operationMax <= 0 {
		return nil, ErrWorkspacePersistenceUnavailable
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ownerID, ok := actor.UserID()
	if !ok {
		return nil, ErrAgentAccessDenied
	}
	if limit < 1 || limit > maxPairedAgentListLimit {
		return nil, ErrInvalidPairedAgentList
	}
	checkedAt := repository.now().UTC()
	if checkedAt.IsZero() {
		return nil, ErrWorkspacePersistenceUnavailable
	}

	operationCtx, cancelOperation := context.WithTimeout(ctx, repository.operationMax)
	defer cancelOperation()
	tx, err := repository.transactions.Begin(operationCtx)
	if err != nil {
		return nil, workspacePersistenceError("begin paired agent list", err)
	}
	if tx == nil {
		return nil, ErrWorkspacePersistenceUnavailable
	}
	defer func() {
		rollbackCtx, cancelRollback := context.WithTimeout(
			context.WithoutCancel(ctx), transactionCleanupTimeout,
		)
		defer cancelRollback()
		rollbackErr := tx.Rollback(rollbackCtx)
		if rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) && err == nil {
			summaries = nil
			err = workspacePersistenceError("rollback paired agent list", rollbackErr)
		}
	}()

	rows, err := tx.Query(operationCtx, `
		SELECT CASE WHEN octet_length(id) = 32 THEN id END,
		       CASE WHEN octet_length(name) BETWEEN 1 AND $2 THEN name END,
		       CASE WHEN octet_length(static_public_key) = 32 THEN static_public_key END,
		       CASE WHEN octet_length(public_key_fingerprint) = $3
		         THEN public_key_fingerprint END,
		       CASE WHEN octet_length(version) BETWEEN 1 AND $4 THEN version END,
		       created_at, last_seen_at, revoked_at
		FROM workspace.agents
		WHERE user_id = $1
		ORDER BY created_at DESC, id
		LIMIT $5`, ownerID.String(), maxAgentNameBytes, canonicalX25519FingerprintLength,
		maxAgentVersionBytes, limit)
	if err != nil {
		return nil, workspacePersistenceError("query paired agent list", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id, name, storedFingerprint, version *string
		var publicKeyBytes []byte
		var createdAt, lastSeenAt, revokedAt pgtype.Timestamptz
		if err := rows.Scan(
			&id, &name, &publicKeyBytes, &storedFingerprint, &version,
			&createdAt, &lastSeenAt, &revokedAt,
		); err != nil {
			return nil, workspacePersistenceError("scan paired agent list", err)
		}
		if id == nil || name == nil || storedFingerprint == nil || version == nil ||
			!createdAt.Valid || createdAt.InfinityModifier != pgtype.Finite || createdAt.Time.IsZero() {
			return nil, fmt.Errorf("%w: corrupt paired agent list row", ErrWorkspacePersistenceUnavailable)
		}
		publicKey, keyErr := cryptox.ParseX25519PublicKey(publicKeyBytes)
		storedAgent, agentErr := NewAgent(
			actor, *id, *name, publicKey, *version, createdAt.Time,
		)
		parsedFingerprint, fingerprintErr := cryptox.ParseX25519Fingerprint(*storedFingerprint)
		computedFingerprint, computedErr := cryptox.FingerprintX25519PublicKey(publicKey)
		if keyErr != nil || agentErr != nil || fingerprintErr != nil || computedErr != nil ||
			!parsedFingerprint.Equal(computedFingerprint) || storedAgent.createdAt.After(checkedAt) {
			return nil, fmt.Errorf("%w: corrupt paired agent list row", ErrWorkspacePersistenceUnavailable)
		}

		summary := PairedAgentSummary{
			ID: storedAgent.id.String(), Name: storedAgent.name, Version: storedAgent.version,
			Fingerprint: *storedFingerprint, CreatedAt: storedAgent.createdAt,
		}
		if lastSeenAt.Valid {
			if lastSeenAt.InfinityModifier != pgtype.Finite || lastSeenAt.Time.IsZero() ||
				lastSeenAt.Time.Before(summary.CreatedAt) || lastSeenAt.Time.After(checkedAt) {
				return nil, fmt.Errorf("%w: corrupt paired agent list row", ErrWorkspacePersistenceUnavailable)
			}
			normalized := lastSeenAt.Time.UTC()
			summary.LastSeenAt = &normalized
		}
		if revokedAt.Valid {
			if revokedAt.InfinityModifier != pgtype.Finite || revokedAt.Time.IsZero() ||
				revokedAt.Time.Before(summary.CreatedAt) || revokedAt.Time.After(checkedAt) {
				return nil, fmt.Errorf("%w: corrupt paired agent list row", ErrWorkspacePersistenceUnavailable)
			}
			normalized := revokedAt.Time.UTC()
			summary.RevokedAt = &normalized
		}
		if !summary.Valid() {
			return nil, fmt.Errorf("%w: corrupt paired agent list row", ErrWorkspacePersistenceUnavailable)
		}
		summaries = append(summaries, summary)
	}
	if err := rows.Err(); err != nil {
		return nil, workspacePersistenceError("iterate paired agent list", err)
	}
	if err := tx.Commit(operationCtx); err != nil {
		return nil, workspacePersistenceError("commit paired agent list", err)
	}
	return summaries, nil
}
