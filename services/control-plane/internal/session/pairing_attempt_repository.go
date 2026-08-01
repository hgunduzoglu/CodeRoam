package session

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"time"

	"github.com/hgunduzoglu/coderoam/packages/go/cryptox"
	"github.com/hgunduzoglu/coderoam/packages/go/ids"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrPairingAttemptAlreadyExists          = errors.New("pairing attempt already exists")
	ErrPairingAttemptUnavailable            = errors.New("pairing attempt unavailable")
	ErrPairingAttemptPersistenceUnavailable = errors.New("pairing attempt persistence unavailable")
)

func (repository *Repository) CreatePairingAttempt(
	ctx context.Context,
	tx pgx.Tx,
	attempt PairingAttempt,
) error {
	if ctx == nil || repository == nil || repository.operationMax <= 0 {
		return ErrPairingAttemptPersistenceUnavailable
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if tx == nil {
		return ErrPairingAttemptPersistenceUnavailable
	}
	if !attempt.validForCreate() {
		return ErrInvalidPairingAttempt
	}
	operationCtx, cancelOperation := context.WithTimeout(ctx, repository.operationMax)
	defer cancelOperation()

	publicKey, err := attempt.agentPublicKey.Bytes()
	if err != nil {
		return ErrInvalidPairingAttempt
	}
	fingerprint, err := attempt.agentFingerprint.String()
	if err != nil {
		return ErrInvalidPairingAttempt
	}
	result, err := tx.Exec(operationCtx, `
		INSERT INTO session.pairing_attempts (
			id, agent_static_public_key, agent_key_fingerprint, agent_display_name,
			agent_version, protocol_version, relay_region, bootstrap_credential_hash,
			expires_at, failed_attempt_count, state, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		attempt.id.String(), publicKey, fingerprint, attempt.agentDisplayName,
		attempt.agentVersion, attempt.protocolVersion, attempt.relayRegion,
		attempt.bootstrapCredentialHash[:], attempt.expiresAt, attempt.failedAttemptCount,
		string(attempt.state), attempt.createdAt, attempt.updatedAt,
	)
	if err != nil {
		var databaseErr *pgconn.PgError
		if errors.As(err, &databaseErr) && databaseErr.Code == "23505" &&
			databaseErr.ConstraintName == "pairing_attempts_pkey" {
			return ErrPairingAttemptAlreadyExists
		}
		return pairingAttemptPersistenceError("create", err)
	}
	if result.RowsAffected() != 1 {
		return fmt.Errorf(
			"%w: create affected %d rows",
			ErrPairingAttemptPersistenceUnavailable,
			result.RowsAffected(),
		)
	}
	return nil
}

func (repository *Repository) LockOpenPairingAttempt(
	ctx context.Context,
	tx pgx.Tx,
	encodedID string,
) (PairingAttempt, error) {
	if ctx == nil || repository == nil || repository.now == nil || repository.operationMax <= 0 {
		return PairingAttempt{}, ErrPairingAttemptPersistenceUnavailable
	}
	if err := ctx.Err(); err != nil {
		return PairingAttempt{}, err
	}
	attemptID, err := ids.Parse(encodedID)
	if err != nil {
		return PairingAttempt{}, fmt.Errorf("%w: id", ErrInvalidPairingAttempt)
	}
	checkedAt := repository.now().UTC()
	if checkedAt.IsZero() {
		return PairingAttempt{}, ErrPairingAttemptPersistenceUnavailable
	}
	if tx == nil {
		return PairingAttempt{}, ErrPairingAttemptPersistenceUnavailable
	}
	operationCtx, cancelOperation := context.WithTimeout(ctx, repository.operationMax)
	defer cancelOperation()

	var publicKeyBytes, bootstrapHash []byte
	var fingerprint, displayName, version, relayRegion string
	var protocolVersion, failedAttemptCount int
	var expiresAt, createdAt, updatedAt time.Time
	err = tx.QueryRow(operationCtx, `
		SELECT agent_static_public_key, agent_key_fingerprint, agent_display_name,
		       agent_version, protocol_version, relay_region, bootstrap_credential_hash,
		       expires_at, failed_attempt_count, created_at, updated_at
		FROM session.pairing_attempts
		WHERE id = $1
		  AND state = 'open'
		  AND created_at <= $2 AND updated_at <= $2 AND expires_at > $2
		  AND failed_attempt_count < $3
		  AND claimed_user_id IS NULL AND device_id IS NULL
		  AND device_display_name IS NULL AND device_platform IS NULL
		  AND device_static_public_key IS NULL AND device_key_fingerprint IS NULL
		  AND claimed_at IS NULL
		  AND mobile_channel_binding IS NULL AND mobile_confirmed_at IS NULL
		  AND agent_channel_binding IS NULL AND agent_confirmed_at IS NULL
		  AND consumed_at IS NULL
		FOR UPDATE`, attemptID.String(), checkedAt, maxPairingAttemptFailures).Scan(
		&publicKeyBytes, &fingerprint, &displayName, &version, &protocolVersion,
		&relayRegion, &bootstrapHash, &expiresAt, &failedAttemptCount, &createdAt, &updatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return PairingAttempt{}, ErrPairingAttemptUnavailable
	}
	if err != nil {
		return PairingAttempt{}, pairingAttemptPersistenceError("lock open", err)
	}
	lockedAt := repository.now().UTC()
	if lockedAt.IsZero() || lockedAt.Before(checkedAt) || createdAt.After(lockedAt) ||
		updatedAt.After(lockedAt) || !expiresAt.After(lockedAt) {
		return PairingAttempt{}, ErrPairingAttemptUnavailable
	}
	publicKey, err := cryptox.ParseX25519PublicKey(publicKeyBytes)
	if err != nil {
		return PairingAttempt{}, ErrPairingAttemptUnavailable
	}
	attempt, err := NewPairingAttempt(PairingAttemptSpec{
		ID: attemptID.String(), AgentPublicKey: publicKey,
		AgentDisplayName: displayName, AgentVersion: version,
		ProtocolVersion: protocolVersion, RelayRegion: relayRegion,
		BootstrapCredentialHash: bootstrapHash, CreatedAt: createdAt, ExpiresAt: expiresAt,
	})
	if err != nil || attempt.agentDisplayName != displayName || attempt.agentVersion != version ||
		failedAttemptCount < 0 || failedAttemptCount >= maxPairingAttemptFailures ||
		updatedAt.Before(createdAt) || updatedAt.After(expiresAt) {
		return PairingAttempt{}, ErrPairingAttemptUnavailable
	}
	storedFingerprint, err := cryptox.ParseX25519Fingerprint(fingerprint)
	if err != nil || !attempt.agentFingerprint.Equal(storedFingerprint) {
		return PairingAttempt{}, ErrPairingAttemptUnavailable
	}
	attempt.failedAttemptCount = failedAttemptCount
	attempt.updatedAt = updatedAt.UTC()
	attempt.lockedAt = lockedAt
	return attempt, nil
}

// authenticateOpenPairingAttempt locks one usable open attempt, compares the supplied
// credential in constant time, and records one bounded failure as a normal transaction
// outcome. The transaction-owning service commits a rejection before exposing it.
func (repository *Repository) authenticateOpenPairingAttempt(
	ctx context.Context,
	tx pgx.Tx,
	encodedID string,
	credential []byte,
) (PairingAttempt, bool, error) {
	attempt, err := repository.LockOpenPairingAttempt(ctx, tx, encodedID)
	if err != nil {
		if errors.Is(err, ErrInvalidPairingAttempt) {
			return PairingAttempt{}, false, ErrPairingAttemptUnavailable
		}
		return PairingAttempt{}, false, err
	}

	candidateHash, credentialErr := HashPairingBootstrapCredential(attempt.id.String(), credential)
	matched := subtle.ConstantTimeCompare(candidateHash[:], attempt.bootstrapCredentialHash[:])
	clear(candidateHash[:])
	if err := ctx.Err(); err != nil {
		return PairingAttempt{}, false, err
	}

	authenticatedAt := repository.now().UTC()
	if authenticatedAt.IsZero() || authenticatedAt.Before(attempt.lockedAt) {
		return PairingAttempt{}, false, ErrPairingAttemptPersistenceUnavailable
	}
	if !attempt.expiresAt.After(authenticatedAt) {
		return PairingAttempt{}, false, ErrPairingAttemptUnavailable
	}
	attempt.lockedAt = authenticatedAt
	if credentialErr == nil && matched == 1 {
		return attempt, true, nil
	}

	failedAt := authenticatedAt.Truncate(time.Microsecond)
	if failedAt.Before(attempt.updatedAt) || !attempt.expiresAt.After(failedAt) {
		return PairingAttempt{}, false, ErrPairingAttemptUnavailable
	}
	operationCtx, cancelOperation := context.WithTimeout(ctx, repository.operationMax)
	defer cancelOperation()

	result, err := tx.Exec(operationCtx, `
		UPDATE session.pairing_attempts
		SET failed_attempt_count = failed_attempt_count + 1, updated_at = $1
		WHERE id = $2
		  AND state = 'open'
		  AND failed_attempt_count = $3
		  AND failed_attempt_count < $4
		  AND created_at <= $1 AND updated_at <= $1 AND expires_at > $1`,
		failedAt, attempt.id.String(), attempt.failedAttemptCount, maxPairingAttemptFailures,
	)
	if err != nil {
		return PairingAttempt{}, false, pairingAttemptPersistenceError("record credential failure", err)
	}
	if result.RowsAffected() != 1 {
		return PairingAttempt{}, false, ErrPairingAttemptUnavailable
	}
	return PairingAttempt{}, false, nil
}

func pairingAttemptPersistenceError(operation string, err error) error {
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return context.DeadlineExceeded
	}
	return fmt.Errorf("%w: %s: %w", ErrPairingAttemptPersistenceUnavailable, operation, err)
}
