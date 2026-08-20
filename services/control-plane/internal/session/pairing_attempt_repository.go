package session

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"time"

	"github.com/hgunduzoglu/coderoam/packages/go/cryptox"
	"github.com/hgunduzoglu/coderoam/packages/go/ids"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/auth"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/device"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
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
			id, agent_id, agent_static_public_key, agent_key_fingerprint, agent_display_name,
			agent_version, protocol_version, relay_region, bootstrap_credential_hash,
			expires_at, failed_attempt_count, state, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
		attempt.id.String(), attempt.agentID.String(), publicKey, fingerprint, attempt.agentDisplayName,
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
	var agentID, fingerprint, displayName, version, relayRegion string
	var protocolVersion, failedAttemptCount int
	var expiresAt, createdAt, updatedAt time.Time
	err = tx.QueryRow(operationCtx, `
		SELECT agent_id, agent_static_public_key, agent_key_fingerprint, agent_display_name,
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
		&agentID, &publicKeyBytes, &fingerprint, &displayName, &version, &protocolVersion,
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
		ID: attemptID.String(), AgentID: agentID, AgentPublicKey: publicKey,
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

// claimOpenPairingAttempt binds one authenticated owner and canonical mobile
// candidate while the usable open attempt remains locked by the caller's transaction.
// An exact retry of a committed claim is idempotent; every mismatch is unavailable.
func (repository *Repository) claimOpenPairingAttempt(
	ctx context.Context,
	tx pgx.Tx,
	encodedID string,
	claim PairingAttemptClaim,
) error {
	if ctx == nil || repository == nil || repository.now == nil || repository.operationMax <= 0 {
		return ErrPairingAttemptPersistenceUnavailable
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if tx == nil {
		return ErrPairingAttemptPersistenceUnavailable
	}
	if !claim.valid() {
		return ErrInvalidPairingAttemptClaim
	}

	attempt, err := repository.LockOpenPairingAttempt(ctx, tx, encodedID)
	if err != nil {
		if errors.Is(err, ErrInvalidPairingAttempt) || errors.Is(err, ErrPairingAttemptUnavailable) {
			return repository.matchClaimedPairingAttempt(ctx, tx, encodedID, claim)
		}
		return err
	}
	if attempt.protocolVersion != claim.protocolVersion ||
		!attempt.agentPublicKey.Equal(claim.expectedAgentPublicKey) ||
		!attempt.agentFingerprint.Equal(claim.expectedAgentFingerprint) {
		return ErrPairingAttemptUnavailable
	}
	claimedAt := attempt.lockedAt.UTC().Truncate(time.Microsecond)
	if claimedAt.Before(attempt.updatedAt) || !attempt.expiresAt.After(claimedAt) {
		return ErrPairingAttemptUnavailable
	}
	publicKey, err := claim.devicePublicKey.Bytes()
	if err != nil {
		return ErrInvalidPairingAttemptClaim
	}
	fingerprint, err := claim.deviceFingerprint.String()
	if err != nil {
		return ErrInvalidPairingAttemptClaim
	}
	operationCtx, cancelOperation := context.WithTimeout(ctx, repository.operationMax)
	defer cancelOperation()

	result, err := tx.Exec(operationCtx, `
		UPDATE session.pairing_attempts
		SET state = 'claimed', claimed_user_id = $1, device_id = $2,
		    device_display_name = $3, device_platform = $4,
		    device_static_public_key = $5, device_key_fingerprint = $6,
		    claimed_at = $7, updated_at = $7
		WHERE id = $8
		  AND state = 'open'
		  AND failed_attempt_count = $9 AND failed_attempt_count < $10
		  AND created_at <= $7 AND updated_at = $11 AND expires_at > $7
		  AND claimed_user_id IS NULL AND device_id IS NULL
		  AND device_display_name IS NULL AND device_platform IS NULL
		  AND device_static_public_key IS NULL AND device_key_fingerprint IS NULL
		  AND claimed_at IS NULL
		  AND mobile_channel_binding IS NULL AND mobile_confirmed_at IS NULL
		  AND agent_channel_binding IS NULL AND agent_confirmed_at IS NULL
		  AND consumed_at IS NULL`,
		claim.ownerID.String(), claim.deviceID.String(), claim.deviceDisplayName,
		string(claim.devicePlatform), publicKey, fingerprint, claimedAt, attempt.id.String(),
		attempt.failedAttemptCount, maxPairingAttemptFailures, attempt.updatedAt,
	)
	if err != nil {
		return pairingAttemptPersistenceError("claim open", err)
	}
	if result.RowsAffected() != 1 {
		return ErrPairingAttemptUnavailable
	}
	return nil
}

type lockedClaimedPairingAttempt struct {
	attempt           PairingAttempt
	claim             PairingAttemptClaim
	claimedAt         time.Time
	mobileBinding     [pairingChannelBindingLen]byte
	mobileConfirmedAt time.Time
	agentBinding      [pairingChannelBindingLen]byte
	agentConfirmedAt  time.Time
	consumedAt        time.Time
	hasMobileBinding  bool
	hasAgentBinding   bool
}

func (repository *Repository) matchClaimedPairingAttempt(
	ctx context.Context,
	tx pgx.Tx,
	encodedID string,
	claim PairingAttemptClaim,
) error {
	locked, err := repository.lockClaimedPairingAttempt(ctx, tx, encodedID)
	if err != nil {
		return err
	}
	if locked.claim.ownerID != claim.ownerID || locked.claim.protocolVersion != claim.protocolVersion ||
		locked.claim.deviceID != claim.deviceID ||
		locked.claim.deviceDisplayName != claim.deviceDisplayName ||
		locked.claim.devicePlatform != claim.devicePlatform ||
		!locked.claim.expectedAgentPublicKey.Equal(claim.expectedAgentPublicKey) ||
		!locked.claim.expectedAgentFingerprint.Equal(claim.expectedAgentFingerprint) ||
		!locked.claim.devicePublicKey.Equal(claim.devicePublicKey) ||
		!locked.claim.deviceFingerprint.Equal(claim.deviceFingerprint) {
		return ErrPairingAttemptUnavailable
	}
	return nil
}

func (repository *Repository) lockClaimedPairingAttempt(
	ctx context.Context,
	tx pgx.Tx,
	encodedID string,
) (lockedClaimedPairingAttempt, error) {
	attemptID, err := ids.Parse(encodedID)
	if err != nil {
		return lockedClaimedPairingAttempt{}, ErrPairingAttemptUnavailable
	}
	checkedAt := repository.now().UTC()
	if checkedAt.IsZero() {
		return lockedClaimedPairingAttempt{}, ErrPairingAttemptPersistenceUnavailable
	}
	operationCtx, cancelOperation := context.WithTimeout(ctx, repository.operationMax)
	defer cancelOperation()

	var agentPublicKey, bootstrapHash, devicePublicKey, mobileBinding, agentBinding []byte
	var agentID, agentFingerprint, agentDisplayName, agentVersion, relayRegion, state string
	var claimedUserID, deviceID, deviceDisplayName, devicePlatform, deviceFingerprint pgtype.Text
	var protocolVersion, failedAttemptCount int
	var expiresAt, createdAt, updatedAt time.Time
	var claimedAt, mobileConfirmedAt, agentConfirmedAt, consumedAt pgtype.Timestamptz
	err = tx.QueryRow(operationCtx, `
		SELECT agent_id, agent_static_public_key, agent_key_fingerprint, agent_display_name,
		       agent_version, protocol_version, relay_region, bootstrap_credential_hash,
		       expires_at, failed_attempt_count, state, created_at, updated_at,
		       claimed_user_id, device_id, device_display_name, device_platform,
		       device_static_public_key, device_key_fingerprint, claimed_at,
		       mobile_channel_binding, mobile_confirmed_at,
		       agent_channel_binding, agent_confirmed_at, consumed_at
		FROM session.pairing_attempts
		WHERE id = $1
		  AND state IN ('claimed', 'confirming', 'consumed')
		  AND created_at <= $2 AND updated_at <= $2
		  AND failed_attempt_count < $3
		  AND (
		    (state IN ('claimed', 'confirming') AND expires_at > $2 AND consumed_at IS NULL)
		    OR (state = 'consumed' AND consumed_at IS NOT NULL AND consumed_at <= $2)
		  )
		FOR UPDATE`, attemptID.String(), checkedAt, maxPairingAttemptFailures).Scan(
		&agentID, &agentPublicKey, &agentFingerprint, &agentDisplayName, &agentVersion,
		&protocolVersion, &relayRegion, &bootstrapHash, &expiresAt, &failedAttemptCount,
		&state, &createdAt, &updatedAt, &claimedUserID, &deviceID, &deviceDisplayName,
		&devicePlatform, &devicePublicKey, &deviceFingerprint, &claimedAt,
		&mobileBinding, &mobileConfirmedAt, &agentBinding, &agentConfirmedAt, &consumedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return lockedClaimedPairingAttempt{}, ErrPairingAttemptUnavailable
	}
	if err != nil {
		return lockedClaimedPairingAttempt{}, pairingAttemptPersistenceError("lock claimed", err)
	}
	if !claimedUserID.Valid || !deviceID.Valid || !deviceDisplayName.Valid || !devicePlatform.Valid ||
		!deviceFingerprint.Valid || !claimedAt.Valid {
		return lockedClaimedPairingAttempt{}, ErrPairingAttemptUnavailable
	}
	lockedAt := repository.now().UTC()
	if lockedAt.IsZero() || lockedAt.Before(checkedAt) || createdAt.After(lockedAt) ||
		updatedAt.After(lockedAt) ||
		(state != string(pairingAttemptStateConsumed) && !expiresAt.After(lockedAt)) {
		return lockedClaimedPairingAttempt{}, ErrPairingAttemptUnavailable
	}

	storedAgentKey, err := cryptox.ParseX25519PublicKey(agentPublicKey)
	if err != nil {
		return lockedClaimedPairingAttempt{}, ErrPairingAttemptUnavailable
	}
	attempt, err := NewPairingAttempt(PairingAttemptSpec{
		ID: attemptID.String(), AgentID: agentID, AgentPublicKey: storedAgentKey,
		AgentDisplayName: agentDisplayName, AgentVersion: agentVersion,
		ProtocolVersion: protocolVersion, RelayRegion: relayRegion,
		BootstrapCredentialHash: bootstrapHash, CreatedAt: createdAt, ExpiresAt: expiresAt,
	})
	if err != nil || attempt.agentDisplayName != agentDisplayName || attempt.agentVersion != agentVersion ||
		failedAttemptCount < 0 || failedAttemptCount >= maxPairingAttemptFailures {
		return lockedClaimedPairingAttempt{}, ErrPairingAttemptUnavailable
	}
	storedAgentFingerprint, err := cryptox.ParseX25519Fingerprint(agentFingerprint)
	if err != nil || !attempt.agentFingerprint.Equal(storedAgentFingerprint) {
		return lockedClaimedPairingAttempt{}, ErrPairingAttemptUnavailable
	}
	storedOwnerID, err := auth.ParseUserID(claimedUserID.String)
	if err != nil {
		return lockedClaimedPairingAttempt{}, ErrPairingAttemptUnavailable
	}
	storedDeviceID, err := ids.Parse(deviceID.String)
	if err != nil {
		return lockedClaimedPairingAttempt{}, ErrPairingAttemptUnavailable
	}
	storedDeviceKey, err := cryptox.ParseX25519PublicKey(devicePublicKey)
	if err != nil || allZero(devicePublicKey) {
		return lockedClaimedPairingAttempt{}, ErrPairingAttemptUnavailable
	}
	storedDeviceFingerprint, err := cryptox.ParseX25519Fingerprint(deviceFingerprint.String)
	if err != nil {
		return lockedClaimedPairingAttempt{}, ErrPairingAttemptUnavailable
	}
	derivedDeviceFingerprint, err := cryptox.FingerprintX25519PublicKey(storedDeviceKey)
	if err != nil || !derivedDeviceFingerprint.Equal(storedDeviceFingerprint) {
		return lockedClaimedPairingAttempt{}, ErrPairingAttemptUnavailable
	}
	normalizedName, ok := normalizedPairingText(
		deviceDisplayName.String, maxPairingDeviceNameRunes, maxPairingDeviceNameBytes,
	)
	if !ok || normalizedName != deviceDisplayName.String ||
		!validPairingDevicePlatform(device.Platform(devicePlatform.String)) ||
		claimedAt.Time.Before(createdAt) || !expiresAt.After(claimedAt.Time) {
		return lockedClaimedPairingAttempt{}, ErrPairingAttemptUnavailable
	}

	attempt.failedAttemptCount = failedAttemptCount
	attempt.state = pairingAttemptState(state)
	attempt.updatedAt = updatedAt.UTC()
	attempt.lockedAt = lockedAt
	locked := lockedClaimedPairingAttempt{
		attempt: attempt,
		claim: PairingAttemptClaim{
			ownerID: storedOwnerID, expectedAgentPublicKey: attempt.agentPublicKey,
			expectedAgentFingerprint: attempt.agentFingerprint, protocolVersion: attempt.protocolVersion,
			deviceID: storedDeviceID, deviceDisplayName: normalizedName,
			devicePlatform: device.Platform(devicePlatform.String), devicePublicKey: storedDeviceKey,
			deviceFingerprint: storedDeviceFingerprint,
		},
		claimedAt: claimedAt.Time.UTC(),
	}
	if attempt.updatedAt.Before(locked.claimedAt) {
		return lockedClaimedPairingAttempt{}, ErrPairingAttemptUnavailable
	}
	if mobileBinding != nil {
		if len(mobileBinding) != pairingChannelBindingLen || allZero(mobileBinding) || !mobileConfirmedAt.Valid {
			return lockedClaimedPairingAttempt{}, ErrPairingAttemptUnavailable
		}
		copy(locked.mobileBinding[:], mobileBinding)
		locked.mobileConfirmedAt = mobileConfirmedAt.Time.UTC()
		locked.hasMobileBinding = true
	} else if mobileConfirmedAt.Valid {
		return lockedClaimedPairingAttempt{}, ErrPairingAttemptUnavailable
	}
	if agentBinding != nil {
		if len(agentBinding) != pairingChannelBindingLen || allZero(agentBinding) || !agentConfirmedAt.Valid {
			return lockedClaimedPairingAttempt{}, ErrPairingAttemptUnavailable
		}
		copy(locked.agentBinding[:], agentBinding)
		locked.agentConfirmedAt = agentConfirmedAt.Time.UTC()
		locked.hasAgentBinding = true
	} else if agentConfirmedAt.Valid {
		return lockedClaimedPairingAttempt{}, ErrPairingAttemptUnavailable
	}
	if locked.hasMobileBinding &&
		(locked.mobileConfirmedAt.Before(locked.claimedAt) ||
			!attempt.expiresAt.After(locked.mobileConfirmedAt) ||
			attempt.updatedAt.Before(locked.mobileConfirmedAt)) {
		return lockedClaimedPairingAttempt{}, ErrPairingAttemptUnavailable
	}
	if locked.hasAgentBinding &&
		(locked.agentConfirmedAt.Before(locked.claimedAt) ||
			!attempt.expiresAt.After(locked.agentConfirmedAt) ||
			attempt.updatedAt.Before(locked.agentConfirmedAt)) {
		return lockedClaimedPairingAttempt{}, ErrPairingAttemptUnavailable
	}
	if consumedAt.Valid {
		locked.consumedAt = consumedAt.Time.UTC()
		if locked.consumedAt.Before(locked.mobileConfirmedAt) ||
			locked.consumedAt.Before(locked.agentConfirmedAt) ||
			locked.consumedAt.After(attempt.expiresAt) ||
			!attempt.updatedAt.Equal(locked.consumedAt) {
			return lockedClaimedPairingAttempt{}, ErrPairingAttemptUnavailable
		}
	}
	if locked.hasMobileBinding && locked.hasAgentBinding &&
		subtle.ConstantTimeCompare(locked.mobileBinding[:], locked.agentBinding[:]) != 1 {
		return lockedClaimedPairingAttempt{}, ErrPairingAttemptUnavailable
	}
	if attempt.state == pairingAttemptStateClaimed {
		if locked.hasMobileBinding || locked.hasAgentBinding ||
			consumedAt.Valid ||
			(attempt.failedAttemptCount == 0 && !attempt.updatedAt.Equal(locked.claimedAt)) {
			return lockedClaimedPairingAttempt{}, ErrPairingAttemptUnavailable
		}
	} else if attempt.state == pairingAttemptStateConfirming {
		if (!locked.hasMobileBinding && !locked.hasAgentBinding) || consumedAt.Valid {
			return lockedClaimedPairingAttempt{}, ErrPairingAttemptUnavailable
		}
	} else if attempt.state != pairingAttemptStateConsumed ||
		!locked.hasMobileBinding || !locked.hasAgentBinding || !consumedAt.Valid {
		return lockedClaimedPairingAttempt{}, ErrPairingAttemptUnavailable
	}
	return locked, nil
}

// confirmMobilePairingAttempt records the authenticated mobile endpoint's exact
// handshake observation under the claimed-attempt row lock. Exact retries are stable.
func (repository *Repository) confirmMobilePairingAttempt(
	ctx context.Context,
	tx pgx.Tx,
	encodedID string,
	confirmation MobilePairingConfirmation,
) error {
	if ctx == nil || repository == nil || repository.now == nil || repository.operationMax <= 0 {
		return ErrPairingAttemptPersistenceUnavailable
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if tx == nil {
		return ErrPairingAttemptPersistenceUnavailable
	}
	if !confirmation.valid() {
		return ErrInvalidPairingAttemptConfirmation
	}
	locked, err := repository.lockClaimedPairingAttempt(ctx, tx, encodedID)
	if err != nil {
		return err
	}
	if locked.claim.ownerID != confirmation.ownerID ||
		locked.attempt.protocolVersion != confirmation.protocolVersion ||
		!locked.attempt.agentPublicKey.Equal(confirmation.observedAgentPublicKey) ||
		!locked.attempt.agentFingerprint.Equal(confirmation.observedAgentFingerprint) {
		return ErrPairingAttemptUnavailable
	}
	if locked.hasMobileBinding {
		if subtle.ConstantTimeCompare(
			locked.mobileBinding[:], confirmation.channelBinding[:],
		) != 1 {
			return ErrPairingAttemptUnavailable
		}
		return nil
	}
	if locked.hasAgentBinding && subtle.ConstantTimeCompare(
		locked.agentBinding[:], confirmation.channelBinding[:],
	) != 1 {
		return ErrPairingAttemptUnavailable
	}
	confirmedAt := locked.attempt.lockedAt.UTC().Truncate(time.Microsecond)
	if confirmedAt.Before(locked.attempt.updatedAt) || !locked.attempt.expiresAt.After(confirmedAt) {
		return ErrPairingAttemptUnavailable
	}
	operationCtx, cancelOperation := context.WithTimeout(ctx, repository.operationMax)
	defer cancelOperation()
	result, err := tx.Exec(operationCtx, `
		UPDATE session.pairing_attempts
		SET state = 'confirming', mobile_channel_binding = $1,
		    mobile_confirmed_at = $2, updated_at = $2
		WHERE id = $3
		  AND state = $4 AND updated_at = $5 AND expires_at > $2
		  AND failed_attempt_count = $6 AND failed_attempt_count < $7
		  AND claimed_user_id = $8
		  AND mobile_channel_binding IS NULL AND mobile_confirmed_at IS NULL
		  AND consumed_at IS NULL`,
		confirmation.channelBinding[:], confirmedAt, locked.attempt.id.String(),
		string(locked.attempt.state), locked.attempt.updatedAt, locked.attempt.failedAttemptCount,
		maxPairingAttemptFailures, confirmation.ownerID.String(),
	)
	if err != nil {
		return pairingAttemptPersistenceError("confirm mobile", err)
	}
	if result.RowsAffected() != 1 {
		return ErrPairingAttemptUnavailable
	}
	return nil
}

// confirmAgentPairingAttempt authenticates the bootstrap credential and records
// the agent endpoint's exact handshake observation under the same row lock.
// A false match is a committed failure-count outcome, not authorization.
func (repository *Repository) confirmAgentPairingAttempt(
	ctx context.Context,
	tx pgx.Tx,
	encodedID string,
	credential []byte,
	confirmation AgentPairingConfirmation,
) (bool, error) {
	if ctx == nil || repository == nil || repository.now == nil || repository.operationMax <= 0 {
		return false, ErrPairingAttemptPersistenceUnavailable
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if tx == nil {
		return false, ErrPairingAttemptPersistenceUnavailable
	}
	if !confirmation.valid() {
		return false, ErrInvalidPairingAttemptConfirmation
	}
	locked, err := repository.lockClaimedPairingAttempt(ctx, tx, encodedID)
	if err != nil {
		return false, err
	}

	candidateHash, credentialErr := HashPairingBootstrapCredential(encodedID, credential)
	matched := subtle.ConstantTimeCompare(candidateHash[:], locked.attempt.bootstrapCredentialHash[:])
	clear(candidateHash[:])
	if err := ctx.Err(); err != nil {
		return false, err
	}
	authenticatedAt := repository.now().UTC()
	if authenticatedAt.IsZero() || authenticatedAt.Before(locked.attempt.lockedAt) {
		return false, ErrPairingAttemptPersistenceUnavailable
	}
	if locked.attempt.state != pairingAttemptStateConsumed &&
		!locked.attempt.expiresAt.After(authenticatedAt) {
		return false, ErrPairingAttemptUnavailable
	}

	if credentialErr != nil || matched != 1 {
		if locked.hasAgentBinding {
			return false, ErrPairingAttemptUnavailable
		}
		failedAt := authenticatedAt.Truncate(time.Microsecond)
		if failedAt.Before(locked.attempt.updatedAt) || !locked.attempt.expiresAt.After(failedAt) {
			return false, ErrPairingAttemptUnavailable
		}
		operationCtx, cancelOperation := context.WithTimeout(ctx, repository.operationMax)
		defer cancelOperation()
		result, updateErr := tx.Exec(operationCtx, `
			UPDATE session.pairing_attempts
			SET failed_attempt_count = failed_attempt_count + 1, updated_at = $1
			WHERE id = $2 AND state = $3
			  AND failed_attempt_count = $4 AND failed_attempt_count < $5
			  AND updated_at = $6 AND expires_at > $1
			  AND claimed_user_id = $7 AND consumed_at IS NULL`,
			failedAt, locked.attempt.id.String(), string(locked.attempt.state),
			locked.attempt.failedAttemptCount, maxPairingAttemptFailures,
			locked.attempt.updatedAt, locked.claim.ownerID.String(),
		)
		if updateErr != nil {
			return false, pairingAttemptPersistenceError("record agent credential failure", updateErr)
		}
		if result.RowsAffected() != 1 {
			return false, ErrPairingAttemptUnavailable
		}
		return false, nil
	}

	if locked.attempt.protocolVersion != confirmation.protocolVersion ||
		!locked.claim.devicePublicKey.Equal(confirmation.observedDevicePublicKey) ||
		!locked.claim.deviceFingerprint.Equal(confirmation.observedDeviceFingerprint) {
		return false, ErrPairingAttemptUnavailable
	}
	if locked.hasAgentBinding {
		if subtle.ConstantTimeCompare(
			locked.agentBinding[:], confirmation.channelBinding[:],
		) != 1 {
			return false, ErrPairingAttemptUnavailable
		}
		return true, nil
	}
	if locked.hasMobileBinding && subtle.ConstantTimeCompare(
		locked.mobileBinding[:], confirmation.channelBinding[:],
	) != 1 {
		return false, ErrPairingAttemptUnavailable
	}

	confirmedAt := authenticatedAt.Truncate(time.Microsecond)
	if confirmedAt.Before(locked.attempt.updatedAt) || !locked.attempt.expiresAt.After(confirmedAt) {
		return false, ErrPairingAttemptUnavailable
	}
	operationCtx, cancelOperation := context.WithTimeout(ctx, repository.operationMax)
	defer cancelOperation()
	result, err := tx.Exec(operationCtx, `
		UPDATE session.pairing_attempts
		SET state = 'confirming', agent_channel_binding = $1,
		    agent_confirmed_at = $2, updated_at = $2
		WHERE id = $3
		  AND state = $4 AND updated_at = $5 AND expires_at > $2
		  AND failed_attempt_count = $6 AND failed_attempt_count < $7
		  AND claimed_user_id = $8
		  AND agent_channel_binding IS NULL AND agent_confirmed_at IS NULL
		  AND (mobile_channel_binding IS NULL OR mobile_channel_binding = $1)
		  AND consumed_at IS NULL`,
		confirmation.channelBinding[:], confirmedAt, locked.attempt.id.String(),
		string(locked.attempt.state), locked.attempt.updatedAt, locked.attempt.failedAttemptCount,
		maxPairingAttemptFailures, locked.claim.ownerID.String(),
	)
	if err != nil {
		return false, pairingAttemptPersistenceError("confirm agent", err)
	}
	if result.RowsAffected() != 1 {
		return false, ErrPairingAttemptUnavailable
	}
	return true, nil
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
