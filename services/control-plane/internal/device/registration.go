package device

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/hgunduzoglu/coderoam/packages/go/cryptox"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/auth"
	"github.com/jackc/pgx/v5"
)

// RegisterPaired persists a pairing-authenticated device in a caller-owned transaction.
// Existing rows are accepted only when the complete active identity is unchanged.
func (repository *Repository) RegisterPaired(
	ctx context.Context,
	tx pgx.Tx,
	ownerID auth.UserID,
	encodedDeviceID string,
	name string,
	platform Platform,
	publicKey cryptox.X25519PublicKey,
	pairedAt time.Time,
) error {
	if ctx == nil || repository == nil || repository.now == nil || repository.operationMax <= 0 {
		return ErrDevicePersistenceUnavailable
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	device, err := newDeviceForOwner(
		ownerID,
		encodedDeviceID,
		name,
		platform,
		publicKey,
		pairedAt.UTC().Truncate(time.Microsecond),
	)
	if err != nil {
		return err
	}
	checkedAt := repository.now().UTC()
	if checkedAt.IsZero() {
		return ErrDevicePersistenceUnavailable
	}
	if device.pairedAt.After(checkedAt) {
		return fmt.Errorf("%w: paired time", ErrInvalidDevice)
	}
	if tx == nil {
		return ErrDevicePersistenceUnavailable
	}
	publicKeyBytes, err := device.publicKey.Bytes()
	if err != nil {
		return fmt.Errorf("%w: public key", ErrInvalidDevice)
	}
	fingerprint, err := cryptox.FingerprintX25519PublicKey(device.publicKey)
	if err != nil {
		return fmt.Errorf("%w: public key", ErrInvalidDevice)
	}
	encodedFingerprint, err := fingerprint.String()
	if err != nil {
		return fmt.Errorf("%w: public key", ErrInvalidDevice)
	}

	operationCtx, cancelOperation := context.WithTimeout(ctx, repository.operationMax)
	defer cancelOperation()
	result, err := tx.Exec(operationCtx, `
		INSERT INTO device.devices (
			id, user_id, name, platform, static_public_key, public_key_fingerprint, paired_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT DO NOTHING`,
		device.id, device.ownerID.String(), device.name, string(device.platform), publicKeyBytes,
		encodedFingerprint, device.pairedAt,
	)
	if err != nil {
		return persistenceError("register paired device", err)
	}
	if result.RowsAffected() == 1 {
		return nil
	}
	if result.RowsAffected() != 0 {
		return fmt.Errorf(
			"%w: paired device insert affected %d rows",
			ErrDevicePersistenceUnavailable,
			result.RowsAffected(),
		)
	}

	rows, err := tx.Query(operationCtx, `
		SELECT id, user_id, name, platform, static_public_key, public_key_fingerprint,
		       paired_at, revoked_at
		FROM device.devices
		WHERE id = $1 OR public_key_fingerprint = $2
		ORDER BY id
		FOR UPDATE`, device.id, encodedFingerprint)
	if err != nil {
		return persistenceError("lock conflicting paired device", err)
	}
	defer rows.Close()

	matched := false
	rowCount := 0
	for rows.Next() {
		rowCount++
		var storedID, storedOwnerID, storedName, storedPlatform, storedFingerprint string
		var storedPublicKey []byte
		var storedPairedAt time.Time
		var revokedAt *time.Time
		if err := rows.Scan(
			&storedID, &storedOwnerID, &storedName, &storedPlatform, &storedPublicKey,
			&storedFingerprint, &storedPairedAt, &revokedAt,
		); err != nil {
			return persistenceError("read conflicting paired device", err)
		}
		matched = storedID == device.id && storedOwnerID == device.ownerID.String() &&
			storedName == device.name && storedPlatform == string(device.platform) &&
			bytes.Equal(storedPublicKey, publicKeyBytes) && storedFingerprint == encodedFingerprint &&
			storedPairedAt.Equal(device.pairedAt) && revokedAt == nil
	}
	if err := rows.Err(); err != nil {
		return persistenceError("iterate conflicting paired devices", err)
	}
	if rowCount != 1 || !matched {
		return ErrDeviceAccessDenied
	}
	return nil
}
