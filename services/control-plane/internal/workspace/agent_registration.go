package workspace

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/hgunduzoglu/coderoam/packages/go/cryptox"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/auth"
	"github.com/jackc/pgx/v5"
)

// RegisterPairedAgent persists a pairing-authenticated agent in a caller-owned transaction.
// Existing rows are accepted only when the complete active identity is unchanged.
func (repository *Repository) RegisterPairedAgent(
	ctx context.Context,
	tx pgx.Tx,
	ownerID auth.UserID,
	encodedAgentID string,
	name string,
	publicKey cryptox.X25519PublicKey,
	version string,
	createdAt time.Time,
) error {
	if ctx == nil || repository == nil || repository.now == nil || repository.operationMax <= 0 {
		return ErrWorkspacePersistenceUnavailable
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	agent, err := newAgentForOwner(
		ownerID,
		encodedAgentID,
		name,
		publicKey,
		version,
		createdAt.UTC().Truncate(time.Microsecond),
	)
	if err != nil {
		return err
	}
	checkedAt := repository.now().UTC()
	if checkedAt.IsZero() {
		return ErrWorkspacePersistenceUnavailable
	}
	if agent.createdAt.After(checkedAt) {
		return fmt.Errorf("%w: creation time", ErrInvalidAgent)
	}
	if tx == nil {
		return ErrWorkspacePersistenceUnavailable
	}
	publicKeyBytes, err := agent.publicKey.Bytes()
	if err != nil {
		return fmt.Errorf("%w: public key", ErrInvalidAgent)
	}
	fingerprint, err := cryptox.FingerprintX25519PublicKey(agent.publicKey)
	if err != nil {
		return fmt.Errorf("%w: public key", ErrInvalidAgent)
	}
	encodedFingerprint, err := fingerprint.String()
	if err != nil {
		return fmt.Errorf("%w: public key", ErrInvalidAgent)
	}

	operationCtx, cancelOperation := context.WithTimeout(ctx, repository.operationMax)
	defer cancelOperation()
	result, err := tx.Exec(operationCtx, `
		INSERT INTO workspace.agents (
			id, user_id, name, static_public_key, public_key_fingerprint, version, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT DO NOTHING`,
		agent.id.String(), agent.ownerID.String(), agent.name, publicKeyBytes,
		encodedFingerprint, agent.version, agent.createdAt,
	)
	if err != nil {
		return workspacePersistenceError("register paired agent", err)
	}
	if result.RowsAffected() == 1 {
		return nil
	}
	if result.RowsAffected() != 0 {
		return fmt.Errorf(
			"%w: paired agent insert affected %d rows",
			ErrWorkspacePersistenceUnavailable,
			result.RowsAffected(),
		)
	}

	rows, err := tx.Query(operationCtx, `
		SELECT id, user_id, name, static_public_key, public_key_fingerprint, version,
		       created_at, revoked_at
		FROM workspace.agents
		WHERE id = $1 OR public_key_fingerprint = $2
		ORDER BY id
		FOR UPDATE`, agent.id.String(), encodedFingerprint)
	if err != nil {
		return workspacePersistenceError("lock conflicting paired agent", err)
	}
	defer rows.Close()

	matched := false
	rowCount := 0
	for rows.Next() {
		rowCount++
		var storedID, storedOwnerID, storedName, storedFingerprint, storedVersion string
		var storedPublicKey []byte
		var storedCreatedAt time.Time
		var revokedAt *time.Time
		if err := rows.Scan(
			&storedID, &storedOwnerID, &storedName, &storedPublicKey, &storedFingerprint,
			&storedVersion, &storedCreatedAt, &revokedAt,
		); err != nil {
			return workspacePersistenceError("read conflicting paired agent", err)
		}
		matched = storedID == agent.id.String() && storedOwnerID == agent.ownerID.String() &&
			storedName == agent.name && bytes.Equal(storedPublicKey, publicKeyBytes) &&
			storedFingerprint == encodedFingerprint && storedVersion == agent.version &&
			storedCreatedAt.Equal(agent.createdAt) && revokedAt == nil
	}
	if err := rows.Err(); err != nil {
		return workspacePersistenceError("iterate conflicting paired agents", err)
	}
	if rowCount != 1 || !matched {
		return ErrAgentAccessDenied
	}
	return nil
}
