package device

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
	maxPairedDeviceListLimit         = 100
	canonicalX25519FingerprintLength = len("x25519-sha256:") + 64
)

var ErrInvalidPairedDeviceList = errors.New("invalid paired device list request")

// PairedDeviceSummary contains owner-visible trust metadata and no key material.
// Revoked devices remain visible so management surfaces can explain their state.
type PairedDeviceSummary struct {
	ID          string
	Name        string
	Platform    Platform
	Fingerprint string
	PairedAt    time.Time
	LastSeenAt  *time.Time
	RevokedAt   *time.Time
}

func (summary PairedDeviceSummary) Valid() bool {
	if _, err := ids.Parse(summary.ID); err != nil {
		return false
	}
	if summary.Name == "" || len(summary.Name) > maxDeviceNameBytes ||
		!utf8.ValidString(summary.Name) || strings.TrimSpace(summary.Name) != summary.Name ||
		strings.ContainsFunc(summary.Name, unicode.IsControl) ||
		utf8.RuneCountInString(summary.Name) > maxDeviceNameRunes {
		return false
	}
	if summary.Platform != PlatformIOS && summary.Platform != PlatformIPadOS &&
		summary.Platform != PlatformAndroid {
		return false
	}
	if _, err := cryptox.ParseX25519Fingerprint(summary.Fingerprint); err != nil ||
		summary.PairedAt.IsZero() {
		return false
	}
	if summary.LastSeenAt != nil &&
		(summary.LastSeenAt.IsZero() || summary.LastSeenAt.Before(summary.PairedAt)) {
		return false
	}
	return summary.RevokedAt == nil ||
		(!summary.RevokedAt.IsZero() && !summary.RevokedAt.Before(summary.PairedAt))
}

// ListPairedDevices returns a bounded owner-scoped management view. Callers
// must still use Authorize for trust decisions.
func (repository *Repository) ListPairedDevices(
	ctx context.Context,
	actor auth.Actor,
	limit int,
) (summaries []PairedDeviceSummary, err error) {
	if ctx == nil || repository == nil || repository.transactions == nil || repository.now == nil ||
		repository.operationMax <= 0 {
		return nil, ErrDevicePersistenceUnavailable
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ownerID, ok := actor.UserID()
	if !ok {
		return nil, ErrDeviceAccessDenied
	}
	if limit < 1 || limit > maxPairedDeviceListLimit {
		return nil, ErrInvalidPairedDeviceList
	}
	checkedAt := repository.now().UTC()
	if checkedAt.IsZero() {
		return nil, ErrDevicePersistenceUnavailable
	}

	operationCtx, cancelOperation := context.WithTimeout(ctx, repository.operationMax)
	defer cancelOperation()
	tx, err := repository.transactions.Begin(operationCtx)
	if err != nil {
		return nil, persistenceError("begin paired device list", err)
	}
	if tx == nil {
		return nil, ErrDevicePersistenceUnavailable
	}
	defer func() {
		rollbackCtx, cancelRollback := context.WithTimeout(
			context.WithoutCancel(ctx), transactionCleanupTimeout,
		)
		defer cancelRollback()
		rollbackErr := tx.Rollback(rollbackCtx)
		if rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) && err == nil {
			summaries = nil
			err = persistenceError("rollback paired device list", rollbackErr)
		}
	}()

	rows, err := tx.Query(operationCtx, `
		SELECT CASE WHEN octet_length(id) = 32 THEN id END,
		       CASE WHEN octet_length(name) BETWEEN 1 AND $2 THEN name END,
		       CASE WHEN platform IN ('ios', 'ipados', 'android') THEN platform END,
		       CASE WHEN octet_length(static_public_key) = 32 THEN static_public_key END,
		       CASE WHEN octet_length(public_key_fingerprint) = $3
		         THEN public_key_fingerprint END,
		       paired_at, last_seen_at, revoked_at
		FROM device.devices
		WHERE user_id = $1
		ORDER BY paired_at DESC, id
		LIMIT $4`, ownerID.String(), maxDeviceNameBytes, canonicalX25519FingerprintLength, limit)
	if err != nil {
		return nil, persistenceError("query paired device list", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id, name, platform, storedFingerprint *string
		var publicKeyBytes []byte
		var pairedAt, lastSeenAt, revokedAt pgtype.Timestamptz
		if err := rows.Scan(
			&id, &name, &platform, &publicKeyBytes, &storedFingerprint,
			&pairedAt, &lastSeenAt, &revokedAt,
		); err != nil {
			return nil, persistenceError("scan paired device list", err)
		}
		if id == nil || name == nil || platform == nil || storedFingerprint == nil ||
			!pairedAt.Valid || pairedAt.InfinityModifier != pgtype.Finite || pairedAt.Time.IsZero() {
			return nil, fmt.Errorf("%w: corrupt paired device list row", ErrDevicePersistenceUnavailable)
		}
		publicKey, keyErr := cryptox.ParseX25519PublicKey(publicKeyBytes)
		storedDevice, deviceErr := NewDevice(
			actor, *id, *name, Platform(*platform), publicKey, pairedAt.Time,
		)
		parsedFingerprint, fingerprintErr := cryptox.ParseX25519Fingerprint(*storedFingerprint)
		computedFingerprint, computedErr := cryptox.FingerprintX25519PublicKey(publicKey)
		if keyErr != nil || deviceErr != nil || fingerprintErr != nil || computedErr != nil ||
			!parsedFingerprint.Equal(computedFingerprint) || storedDevice.pairedAt.After(checkedAt) {
			return nil, fmt.Errorf("%w: corrupt paired device list row", ErrDevicePersistenceUnavailable)
		}

		summary := PairedDeviceSummary{
			ID: storedDevice.id, Name: storedDevice.name, Platform: storedDevice.platform,
			Fingerprint: *storedFingerprint, PairedAt: storedDevice.pairedAt,
		}
		if lastSeenAt.Valid {
			if lastSeenAt.InfinityModifier != pgtype.Finite || lastSeenAt.Time.IsZero() ||
				lastSeenAt.Time.Before(summary.PairedAt) || lastSeenAt.Time.After(checkedAt) {
				return nil, fmt.Errorf("%w: corrupt paired device list row", ErrDevicePersistenceUnavailable)
			}
			normalized := lastSeenAt.Time.UTC()
			summary.LastSeenAt = &normalized
		}
		if revokedAt.Valid {
			if revokedAt.InfinityModifier != pgtype.Finite || revokedAt.Time.IsZero() ||
				revokedAt.Time.Before(summary.PairedAt) || revokedAt.Time.After(checkedAt) {
				return nil, fmt.Errorf("%w: corrupt paired device list row", ErrDevicePersistenceUnavailable)
			}
			normalized := revokedAt.Time.UTC()
			summary.RevokedAt = &normalized
		}
		if !summary.Valid() {
			return nil, fmt.Errorf("%w: corrupt paired device list row", ErrDevicePersistenceUnavailable)
		}
		summaries = append(summaries, summary)
	}
	if err := rows.Err(); err != nil {
		return nil, persistenceError("iterate paired device list", err)
	}
	if err := tx.Commit(operationCtx); err != nil {
		return nil, persistenceError("commit paired device list", err)
	}
	return summaries, nil
}
