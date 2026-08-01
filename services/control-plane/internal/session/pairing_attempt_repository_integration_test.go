package session

import (
	"bytes"
	"context"
	"errors"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hgunduzoglu/coderoam/packages/go/cryptox"
	"github.com/hgunduzoglu/coderoam/packages/go/ids"
	"github.com/hgunduzoglu/coderoam/packages/go/postgresx"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/device"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPairingAttemptRepositoryIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_TEST_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_TEST_DSN is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	pool, err := postgresx.OpenPool(ctx, dsn)
	if err != nil {
		t.Fatalf("OpenPool() error = %v", err)
	}
	t.Cleanup(pool.Close)
	applySessionIntegrationMigration(t, ctx, pool)
	repository := NewRepository()
	createdAt := time.Date(2026, time.July, 31, 17, 0, 0, 0, time.UTC)

	rolledBack := newPairingAttemptRepositoryFixture(t, createdAt)
	deletePairingAttemptFixture(t, pool, rolledBack.id.String())
	rollbackTx := beginSessionIntegrationTx(t, ctx, pool, "pairing attempt rollback")
	if err := repository.CreatePairingAttempt(ctx, rollbackTx, rolledBack); err != nil {
		t.Fatalf("CreatePairingAttempt(rollback) error = %v", err)
	}
	assertStoredPairingAttempt(t, ctx, rollbackTx, rolledBack)
	assertPairingAttemptMissing(t, ctx, pool, rolledBack.id.String())
	rollbackSessionIntegrationTx(t, rollbackTx, "pairing attempt rollback")
	assertPairingAttemptMissing(t, ctx, pool, rolledBack.id.String())

	committed := newPairingAttemptRepositoryFixture(t, createdAt.Add(time.Minute))
	deletePairingAttemptFixture(t, pool, committed.id.String())
	t.Cleanup(func() { deletePairingAttemptFixture(t, pool, committed.id.String()) })
	commitTx := beginSessionIntegrationTx(t, ctx, pool, "pairing attempt commit")
	if err := repository.CreatePairingAttempt(ctx, commitTx, committed); err != nil {
		t.Fatalf("CreatePairingAttempt(commit) error = %v", err)
	}
	assertPairingAttemptMissing(t, ctx, pool, committed.id.String())
	if err := commitTx.Commit(ctx); err != nil {
		t.Fatalf("commit pairing attempt: %v", err)
	}
	assertStoredPairingAttempt(t, ctx, pool, committed)
	assertPairingBootstrapCredentialAuthentication(t, ctx, pool, repository, createdAt.Add(2*time.Minute))
	assertPairingAttemptClaimTransition(t, ctx, pool, repository, createdAt.Add(3*time.Minute))
	assertConcurrentPairingAttemptClaims(t, ctx, pool, repository, createdAt.Add(4*time.Minute))
	assertPairingAttemptClaimCommitReconciliation(t, ctx, pool, repository, createdAt.Add(5*time.Minute))

	duplicateTx := beginSessionIntegrationTx(t, ctx, pool, "pairing attempt duplicate")
	if err := repository.CreatePairingAttempt(
		ctx, duplicateTx, committed,
	); !errors.Is(err, ErrPairingAttemptAlreadyExists) {
		t.Fatalf("CreatePairingAttempt(duplicate) error = %v", err)
	}
	rollbackSessionIntegrationTx(t, duplicateTx, "pairing attempt duplicate")

	invalid := newPairingAttemptRepositoryFixture(t, createdAt.Add(2*time.Minute))
	invalid.agentDisplayName = " M3 agent"
	invalidTx := beginSessionIntegrationTx(t, ctx, pool, "pairing attempt invalid")
	if err := repository.CreatePairingAttempt(ctx, invalidTx, invalid); !errors.Is(err, ErrInvalidPairingAttempt) {
		t.Fatalf("CreatePairingAttempt(invalid) error = %v", err)
	}
	invalid.agentDisplayName = "M3 agent"
	invalid.relayRegion = "eu--test"
	if err := repository.CreatePairingAttempt(ctx, invalidTx, invalid); !errors.Is(err, ErrInvalidPairingAttempt) {
		t.Fatalf("CreatePairingAttempt(noncanonical relay region) error = %v", err)
	}
	invalid.relayRegion = "eu-test-1"
	invalid.createdAt = invalid.createdAt.Truncate(time.Microsecond)
	invalid.expiresAt = invalid.createdAt.Add(time.Nanosecond)
	invalid.updatedAt = invalid.createdAt
	if err := repository.CreatePairingAttempt(ctx, invalidTx, invalid); !errors.Is(err, ErrInvalidPairingAttempt) {
		t.Fatalf("CreatePairingAttempt(sub-microsecond lifetime) error = %v", err)
	}
	if _, err := invalidTx.Exec(ctx, `SELECT 1`); err != nil {
		t.Fatalf("invalid pairing attempt made caller transaction unusable: %v", err)
	}
	rollbackSessionIntegrationTx(t, invalidTx, "pairing attempt invalid")

	checkedAt := committed.createdAt.Add(time.Minute)
	repository.now = func() time.Time { return checkedAt }
	lockingTx := beginSessionIntegrationTx(t, ctx, pool, "pairing attempt locking")
	locked, err := repository.LockOpenPairingAttempt(ctx, lockingTx, committed.id.String())
	if err != nil {
		t.Fatalf("LockOpenPairingAttempt(locking) error = %v", err)
	}
	if locked.id != committed.id || !locked.agentPublicKey.Equal(committed.agentPublicKey) ||
		!locked.agentFingerprint.Equal(committed.agentFingerprint) ||
		locked.failedAttemptCount != 0 || locked.state != pairingAttemptStateOpen {
		t.Fatal("LockOpenPairingAttempt() did not restore canonical open-attempt metadata")
	}

	boundedTx := beginSessionIntegrationTx(t, ctx, pool, "pairing attempt bounded")
	repository.operationMax = 100 * time.Millisecond
	if _, err := repository.LockOpenPairingAttempt(
		context.Background(), boundedTx, committed.id.String(),
	); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("LockOpenPairingAttempt(contended) error = %v, want deadline exceeded", err)
	}
	rollbackSessionIntegrationTx(t, boundedTx, "pairing attempt bounded")
	rollbackSessionIntegrationTx(t, lockingTx, "pairing attempt locking")

	repository.operationMax = repositoryOperationTimeout
	retryTx := beginSessionIntegrationTx(t, ctx, pool, "pairing attempt retry")
	if _, err := repository.LockOpenPairingAttempt(
		ctx, retryTx, committed.id.String(),
	); err != nil {
		t.Fatalf("LockOpenPairingAttempt(after contention) error = %v", err)
	}
	rollbackSessionIntegrationTx(t, retryTx, "pairing attempt retry")
	assertPairingAttemptExpiresWhileWaiting(t, ctx, pool, repository, committed)

	assertOpenPairingAttemptUnavailable(t, ctx, pool, repository, committed.id.String(), committed.expiresAt)
	assertOpenPairingAttemptUnavailable(t, ctx, pool, repository, committed.id.String(), committed.createdAt.Add(-time.Nanosecond))
	missingID, err := ids.New()
	if err != nil {
		t.Fatalf("ids.New(missing pairing attempt) error = %v", err)
	}
	assertOpenPairingAttemptUnavailable(t, ctx, pool, repository, missingID.String(), checkedAt)

	if _, err := pool.Exec(ctx, `
		UPDATE session.pairing_attempts
		SET failed_attempt_count = $1
		WHERE id = $2`, maxPairingAttemptFailures, committed.id.String()); err != nil {
		t.Fatalf("exhaust pairing attempt fixture: %v", err)
	}
	assertOpenPairingAttemptUnavailable(t, ctx, pool, repository, committed.id.String(), checkedAt)

	if _, err := pool.Exec(ctx, `
		UPDATE session.pairing_attempts
		SET failed_attempt_count = 0, agent_display_name = $1
		WHERE id = $2`, "\u00a0"+committed.agentDisplayName+"\u00a0", committed.id.String()); err != nil {
		t.Fatalf("persist noncanonical pairing-attempt name fixture: %v", err)
	}
	assertOpenPairingAttemptUnavailable(t, ctx, pool, repository, committed.id.String(), checkedAt)
	if _, err := pool.Exec(ctx, `
		UPDATE session.pairing_attempts
		SET agent_display_name = $1
		WHERE id = $2`, "agent\u202ename", committed.id.String()); err != nil {
		t.Fatalf("persist bidi pairing-attempt name fixture: %v", err)
	}
	assertOpenPairingAttemptUnavailable(t, ctx, pool, repository, committed.id.String(), checkedAt)
}

func assertPairingBootstrapCredentialAuthentication(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	repository *Repository,
	createdAt time.Time,
) {
	t.Helper()
	attempt := newPairingAttemptRepositoryFixture(t, createdAt)
	deletePairingAttemptFixture(t, pool, attempt.id.String())
	t.Cleanup(func() { deletePairingAttemptFixture(t, pool, attempt.id.String()) })

	createTx := beginSessionIntegrationTx(t, ctx, pool, "pairing credential create")
	if err := repository.CreatePairingAttempt(ctx, createTx, attempt); err != nil {
		t.Fatalf("CreatePairingAttempt(credential fixture) error = %v", err)
	}
	if err := createTx.Commit(ctx); err != nil {
		t.Fatalf("commit pairing credential fixture: %v", err)
	}
	service, err := newPairingAttemptService(pool, repository)
	if err != nil {
		t.Fatalf("newPairingAttemptService() error = %v", err)
	}

	checkedAt := createdAt.Add(time.Minute)
	repository.now = func() time.Time { return checkedAt }
	successTx := beginSessionIntegrationTx(t, ctx, pool, "pairing credential success")
	authenticated, matched, err := repository.authenticateOpenPairingAttempt(
		ctx, successTx, attempt.id.String(), validPairingBootstrapCredential(),
	)
	if err != nil || !matched {
		t.Fatalf("authenticateOpenPairingAttempt(valid) matched = %t, error = %v", matched, err)
	}
	if authenticated.id != attempt.id || authenticated.failedAttemptCount != 0 ||
		!authenticated.lockedAt.Equal(checkedAt) {
		t.Fatal("AuthenticateOpenPairingAttempt(valid) returned inconsistent locked metadata")
	}
	assertPairingAttemptFailureCount(t, ctx, successTx, attempt.id.String(), 0)
	rollbackSessionIntegrationTx(t, successTx, "pairing credential success")

	wrongCredential := bytes.Repeat([]byte{0x6b}, pairingBootstrapCredentialLen)
	rolledBackTx := beginSessionIntegrationTx(t, ctx, pool, "pairing credential failure rollback")
	if _, matched, err := repository.authenticateOpenPairingAttempt(
		ctx, rolledBackTx, attempt.id.String(), wrongCredential,
	); err != nil || matched {
		t.Fatalf("authenticateOpenPairingAttempt(wrong rollback) matched = %t, error = %v", matched, err)
	}
	assertPairingAttemptFailureCount(t, ctx, rolledBackTx, attempt.id.String(), 1)
	assertPairingAttemptFailureCount(t, ctx, pool, attempt.id.String(), 0)
	rollbackSessionIntegrationTx(t, rolledBackTx, "pairing credential failure rollback")
	assertPairingAttemptFailureCount(t, ctx, pool, attempt.id.String(), 0)

	if err := service.authenticateBootstrapCredential(
		ctx, attempt.id.String(), nil,
	); !errors.Is(err, ErrPairingAttemptUnavailable) {
		t.Fatalf("authenticateBootstrapCredential(malformed) error = %v", err)
	}
	assertPairingAttemptFailureCount(t, ctx, pool, attempt.id.String(), 1)

	retryTx := beginSessionIntegrationTx(t, ctx, pool, "pairing credential valid retry")
	authenticated, matched, err = repository.authenticateOpenPairingAttempt(
		ctx, retryTx, attempt.id.String(), validPairingBootstrapCredential(),
	)
	if err != nil || !matched {
		t.Fatalf("authenticateOpenPairingAttempt(valid retry) matched = %t, error = %v", matched, err)
	}
	if authenticated.failedAttemptCount != 1 {
		t.Fatalf("authenticated failed count = %d, want 1", authenticated.failedAttemptCount)
	}
	rollbackSessionIntegrationTx(t, retryTx, "pairing credential valid retry")

	for wantFailures := 2; wantFailures <= maxPairingAttemptFailures; wantFailures++ {
		if err := service.authenticateBootstrapCredential(
			ctx, attempt.id.String(), wrongCredential,
		); !errors.Is(err, ErrPairingAttemptUnavailable) {
			t.Fatalf("authenticateBootstrapCredential(exhaust %d) error = %v", wantFailures, err)
		}
		assertPairingAttemptFailureCount(t, ctx, pool, attempt.id.String(), wantFailures)
	}
	assertPairingAttemptFailureCount(t, ctx, pool, attempt.id.String(), maxPairingAttemptFailures)

	if err := service.authenticateBootstrapCredential(
		ctx, attempt.id.String(), validPairingBootstrapCredential(),
	); !errors.Is(err, ErrPairingAttemptUnavailable) {
		t.Fatalf("authenticateBootstrapCredential(exhausted valid) error = %v", err)
	}

	assertPairingBootstrapCredentialTimeBoundary(t, ctx, pool, repository, createdAt.Add(time.Minute))
}

func assertPairingBootstrapCredentialTimeBoundary(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	repository *Repository,
	createdAt time.Time,
) {
	t.Helper()
	attempt := newPairingAttemptRepositoryFixture(t, createdAt)
	deletePairingAttemptFixture(t, pool, attempt.id.String())
	t.Cleanup(func() { deletePairingAttemptFixture(t, pool, attempt.id.String()) })
	createTx := beginSessionIntegrationTx(t, ctx, pool, "pairing credential time create")
	if err := repository.CreatePairingAttempt(ctx, createTx, attempt); err != nil {
		t.Fatalf("CreatePairingAttempt(time fixture) error = %v", err)
	}
	if err := createTx.Commit(ctx); err != nil {
		t.Fatalf("commit pairing credential time fixture: %v", err)
	}

	var calls atomic.Int64
	repository.now = func() time.Time {
		if calls.Add(1) <= 2 {
			return attempt.expiresAt.Add(-time.Microsecond)
		}
		return attempt.expiresAt
	}
	expiryTx := beginSessionIntegrationTx(t, ctx, pool, "pairing credential exact expiry")
	if _, matched, err := repository.authenticateOpenPairingAttempt(
		ctx, expiryTx, attempt.id.String(), validPairingBootstrapCredential(),
	); !errors.Is(err, ErrPairingAttemptUnavailable) || matched {
		t.Fatalf("authenticateOpenPairingAttempt(exact expiry) matched = %t, error = %v", matched, err)
	}
	assertPairingAttemptFailureCount(t, ctx, expiryTx, attempt.id.String(), 0)
	rollbackSessionIntegrationTx(t, expiryTx, "pairing credential exact expiry")

	calls.Store(0)
	lockedAt := attempt.createdAt.Add(time.Minute)
	repository.now = func() time.Time {
		if calls.Add(1) <= 2 {
			return lockedAt
		}
		return lockedAt.Add(-time.Microsecond)
	}
	backwardTx := beginSessionIntegrationTx(t, ctx, pool, "pairing credential backward clock")
	if _, matched, err := repository.authenticateOpenPairingAttempt(
		ctx, backwardTx, attempt.id.String(), validPairingBootstrapCredential(),
	); !errors.Is(err, ErrPairingAttemptPersistenceUnavailable) || matched {
		t.Fatalf("authenticateOpenPairingAttempt(backward clock) matched = %t, error = %v", matched, err)
	}
	assertPairingAttemptFailureCount(t, ctx, backwardTx, attempt.id.String(), 0)
	rollbackSessionIntegrationTx(t, backwardTx, "pairing credential backward clock")
}

func assertPairingAttemptClaimTransition(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	repository *Repository,
	createdAt time.Time,
) {
	t.Helper()
	attempt := newPairingAttemptRepositoryFixture(t, createdAt)
	deletePairingAttemptFixture(t, pool, attempt.id.String())
	t.Cleanup(func() { deletePairingAttemptFixture(t, pool, attempt.id.String()) })
	createTx := beginSessionIntegrationTx(t, ctx, pool, "pairing claim create")
	if err := repository.CreatePairingAttempt(ctx, createTx, attempt); err != nil {
		t.Fatalf("CreatePairingAttempt(claim fixture) error = %v", err)
	}
	if err := createTx.Commit(ctx); err != nil {
		t.Fatalf("commit pairing claim fixture: %v", err)
	}
	claim, err := NewPairingAttemptClaim(validPairingAttemptClaimSpec(t))
	if err != nil {
		t.Fatalf("NewPairingAttemptClaim() error = %v", err)
	}
	service, err := newPairingAttemptService(pool, repository)
	if err != nil {
		t.Fatalf("newPairingAttemptService() error = %v", err)
	}
	claimedAt := createdAt.Add(time.Minute)
	repository.now = func() time.Time { return claimedAt }

	rollbackTx := beginSessionIntegrationTx(t, ctx, pool, "pairing claim rollback")
	if err := repository.claimOpenPairingAttempt(
		ctx, rollbackTx, attempt.id.String(), claim,
	); err != nil {
		t.Fatalf("claimOpenPairingAttempt(rollback) error = %v", err)
	}
	assertPairingAttemptClaimed(t, ctx, rollbackTx, attempt.id.String(), claim, claimedAt)
	assertStoredPairingAttempt(t, ctx, pool, attempt)
	rollbackSessionIntegrationTx(t, rollbackTx, "pairing claim rollback")
	assertStoredPairingAttempt(t, ctx, pool, attempt)

	if err := service.claim(ctx, attempt.id.String(), claim); err != nil {
		t.Fatalf("claim() error = %v", err)
	}
	assertPairingAttemptClaimed(t, ctx, pool, attempt.id.String(), claim, claimedAt)

	repository.now = func() time.Time { return claimedAt.Add(time.Second) }
	if err := service.claim(ctx, attempt.id.String(), claim); err != nil {
		t.Fatalf("claim(idempotent retry) error = %v", err)
	}
	assertPairingAttemptClaimed(t, ctx, pool, attempt.id.String(), claim, claimedAt)

	conflicts := map[string]func(*PairingAttemptClaimSpec){
		"foreign owner": func(spec *PairingAttemptClaimSpec) {
			spec.Actor = newSessionTestActor(t, "2123456789abcdef0123456789abcdef")
		},
		"different expected agent key": func(spec *PairingAttemptClaimSpec) {
			key, parseErr := cryptox.ParseX25519PublicKey(bytes.Repeat([]byte{0x72}, 32))
			if parseErr != nil {
				t.Fatalf("ParseX25519PublicKey(agent conflict) error = %v", parseErr)
			}
			spec.ExpectedAgentPublicKey = key
		},
		"different device id": func(spec *PairingAttemptClaimSpec) {
			spec.DeviceID = "3123456789abcdef0123456789abcdef"
		},
		"different display name": func(spec *PairingAttemptClaimSpec) {
			spec.DeviceDisplayName = "Other phone"
		},
		"different platform": func(spec *PairingAttemptClaimSpec) {
			spec.DevicePlatform = device.PlatformAndroid
		},
		"different public key": func(spec *PairingAttemptClaimSpec) {
			key, parseErr := cryptox.ParseX25519PublicKey(bytes.Repeat([]byte{0x71}, 32))
			if parseErr != nil {
				t.Fatalf("ParseX25519PublicKey(conflict) error = %v", parseErr)
			}
			spec.DevicePublicKey = key
		},
	}
	for name, mutate := range conflicts {
		t.Run(name, func(t *testing.T) {
			spec := validPairingAttemptClaimSpec(t)
			mutate(&spec)
			conflict, claimErr := NewPairingAttemptClaim(spec)
			if claimErr != nil {
				t.Fatalf("NewPairingAttemptClaim(conflict) error = %v", claimErr)
			}
			if claimErr := service.claim(
				ctx, attempt.id.String(), conflict,
			); !errors.Is(claimErr, ErrPairingAttemptUnavailable) {
				t.Fatalf("claim(%s) error = %v", name, claimErr)
			}
		})
	}
	corruptTx := beginSessionIntegrationTx(t, ctx, pool, "pairing claim corrupt nullable metadata")
	if _, err := corruptTx.Exec(ctx, `
		ALTER TABLE session.pairing_attempts
		  DROP CONSTRAINT pairing_attempts_claim_shape,
		  DROP CONSTRAINT pairing_attempts_state_shape`,
	); err != nil {
		t.Fatalf("drop pairing claim constraints: %v", err)
	}
	if _, err := corruptTx.Exec(ctx, `
		UPDATE session.pairing_attempts SET claimed_user_id = NULL WHERE id = $1`,
		attempt.id.String(),
	); err != nil {
		t.Fatalf("corrupt claimed pairing-attempt metadata: %v", err)
	}
	if claimErr := repository.claimOpenPairingAttempt(
		ctx, corruptTx, attempt.id.String(), claim,
	); !errors.Is(claimErr, ErrPairingAttemptUnavailable) {
		t.Fatalf("claim(corrupt nullable metadata) error = %v", claimErr)
	}
	rollbackSessionIntegrationTx(t, corruptTx, "pairing claim corrupt nullable metadata")
	if claimErr := service.claim(ctx, "invalid", claim); !errors.Is(
		claimErr, ErrPairingAttemptUnavailable,
	) {
		t.Fatalf("claim(invalid id) error = %v", claimErr)
	}
	repository.now = func() time.Time { return attempt.expiresAt }
	if claimErr := service.claim(ctx, attempt.id.String(), claim); !errors.Is(
		claimErr, ErrPairingAttemptUnavailable,
	) {
		t.Fatalf("claim(expired idempotent retry) error = %v", claimErr)
	}
	assertPairingAttemptClaimed(t, ctx, pool, attempt.id.String(), claim, claimedAt)
}

func assertConcurrentPairingAttemptClaims(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	repository *Repository,
	createdAt time.Time,
) {
	t.Helper()
	attempt := newPairingAttemptRepositoryFixture(t, createdAt)
	deletePairingAttemptFixture(t, pool, attempt.id.String())
	t.Cleanup(func() { deletePairingAttemptFixture(t, pool, attempt.id.String()) })
	createTx := beginSessionIntegrationTx(t, ctx, pool, "concurrent pairing claim create")
	if err := repository.CreatePairingAttempt(ctx, createTx, attempt); err != nil {
		t.Fatalf("CreatePairingAttempt(concurrent claim fixture) error = %v", err)
	}
	if err := createTx.Commit(ctx); err != nil {
		t.Fatalf("commit concurrent pairing claim fixture: %v", err)
	}
	winner, err := NewPairingAttemptClaim(validPairingAttemptClaimSpec(t))
	if err != nil {
		t.Fatalf("NewPairingAttemptClaim(winner) error = %v", err)
	}
	foreignSpec := validPairingAttemptClaimSpec(t)
	foreignSpec.Actor = newSessionTestActor(t, "2123456789abcdef0123456789abcdef")
	foreign, err := NewPairingAttemptClaim(foreignSpec)
	if err != nil {
		t.Fatalf("NewPairingAttemptClaim(foreign) error = %v", err)
	}
	claimedAt := createdAt.Add(time.Minute)
	repository.now = func() time.Time { return claimedAt }

	winnerTx := beginSessionIntegrationTx(t, ctx, pool, "concurrent pairing claim winner")
	if err := repository.claimOpenPairingAttempt(
		ctx, winnerTx, attempt.id.String(), winner,
	); err != nil {
		t.Fatalf("claimOpenPairingAttempt(winner) error = %v", err)
	}
	exactTx := beginSessionIntegrationTx(t, ctx, pool, "concurrent pairing claim exact retry")
	exactResult := make(chan error, 1)
	go func() {
		exactResult <- repository.claimOpenPairingAttempt(ctx, exactTx, attempt.id.String(), winner)
	}()
	waitForPairingAttemptRowLock(t, ctx, pool, exactTx.Conn().PgConn().PID())

	foreignTx := beginSessionIntegrationTx(t, ctx, pool, "concurrent pairing claim foreign retry")
	foreignResult := make(chan error, 1)
	go func() {
		foreignResult <- repository.claimOpenPairingAttempt(ctx, foreignTx, attempt.id.String(), foreign)
	}()
	waitForPairingAttemptRowLock(t, ctx, pool, foreignTx.Conn().PgConn().PID())
	if err := winnerTx.Commit(ctx); err != nil {
		t.Fatalf("commit concurrent pairing claim winner: %v", err)
	}
	if err := <-exactResult; err != nil {
		t.Fatalf("claimOpenPairingAttempt(concurrent exact retry) error = %v", err)
	}
	if err := exactTx.Commit(ctx); err != nil {
		t.Fatalf("commit concurrent exact retry: %v", err)
	}
	if err := <-foreignResult; !errors.Is(err, ErrPairingAttemptUnavailable) {
		t.Fatalf("claimOpenPairingAttempt(concurrent foreign retry) error = %v", err)
	}
	rollbackSessionIntegrationTx(t, foreignTx, "concurrent pairing claim foreign retry")
	assertPairingAttemptClaimed(t, ctx, pool, attempt.id.String(), winner, claimedAt)
}

func assertPairingAttemptClaimCommitReconciliation(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	repository *Repository,
	createdAt time.Time,
) {
	t.Helper()
	attempt := newPairingAttemptRepositoryFixture(t, createdAt)
	deletePairingAttemptFixture(t, pool, attempt.id.String())
	t.Cleanup(func() { deletePairingAttemptFixture(t, pool, attempt.id.String()) })
	createTx := beginSessionIntegrationTx(t, ctx, pool, "ambiguous pairing claim create")
	if err := repository.CreatePairingAttempt(ctx, createTx, attempt); err != nil {
		t.Fatalf("CreatePairingAttempt(ambiguous claim fixture) error = %v", err)
	}
	if err := createTx.Commit(ctx); err != nil {
		t.Fatalf("commit ambiguous pairing claim fixture: %v", err)
	}
	claim, err := NewPairingAttemptClaim(validPairingAttemptClaimSpec(t))
	if err != nil {
		t.Fatalf("NewPairingAttemptClaim(ambiguous) error = %v", err)
	}
	claimedAt := createdAt.Add(time.Minute)
	repository.now = func() time.Time { return claimedAt }
	ackErr := errors.New("claim commit acknowledgement lost")
	ambiguousService, err := newPairingAttemptService(
		&commitAcknowledgementLostStarter{pool: pool, err: ackErr}, repository,
	)
	if err != nil {
		t.Fatalf("newPairingAttemptService(ambiguous) error = %v", err)
	}
	if claimErr := ambiguousService.claim(
		ctx, attempt.id.String(), claim,
	); !errors.Is(claimErr, ErrPairingAttemptCommitOutcomeUnknown) || !errors.Is(claimErr, ackErr) {
		t.Fatalf("claim(ambiguous committed outcome) error = %v", claimErr)
	}
	assertPairingAttemptClaimed(t, ctx, pool, attempt.id.String(), claim, claimedAt)

	service, err := newPairingAttemptService(pool, repository)
	if err != nil {
		t.Fatalf("newPairingAttemptService(reconcile) error = %v", err)
	}
	if err := service.claim(ctx, attempt.id.String(), claim); err != nil {
		t.Fatalf("claim(reconcile exact) error = %v", err)
	}
	foreignSpec := validPairingAttemptClaimSpec(t)
	foreignSpec.Actor = newSessionTestActor(t, "2123456789abcdef0123456789abcdef")
	foreign, err := NewPairingAttemptClaim(foreignSpec)
	if err != nil {
		t.Fatalf("NewPairingAttemptClaim(reconcile foreign) error = %v", err)
	}
	if claimErr := service.claim(
		ctx, attempt.id.String(), foreign,
	); !errors.Is(claimErr, ErrPairingAttemptUnavailable) {
		t.Fatalf("claim(reconcile foreign) error = %v", claimErr)
	}
	assertPairingAttemptClaimed(t, ctx, pool, attempt.id.String(), claim, claimedAt)
}

func assertPairingAttemptExpiresWhileWaiting(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	repository *Repository,
	attempt PairingAttempt,
) {
	t.Helper()
	var clock atomic.Int64
	clock.Store(attempt.expiresAt.Add(-time.Second).UnixNano())
	repository.now = func() time.Time { return time.Unix(0, clock.Load()).UTC() }

	lockingTx := beginSessionIntegrationTx(t, ctx, pool, "pairing attempt expiry lock")
	if _, err := repository.LockOpenPairingAttempt(ctx, lockingTx, attempt.id.String()); err != nil {
		t.Fatalf("LockOpenPairingAttempt(expiry lock) error = %v", err)
	}
	waitingTx := beginSessionIntegrationTx(t, ctx, pool, "pairing attempt expiry waiter")
	result := make(chan error, 1)
	go func() {
		_, err := repository.LockOpenPairingAttempt(ctx, waitingTx, attempt.id.String())
		result <- err
	}()
	waitForPairingAttemptRowLock(t, ctx, pool, waitingTx.Conn().PgConn().PID())
	clock.Store(attempt.expiresAt.UnixNano())
	rollbackSessionIntegrationTx(t, lockingTx, "pairing attempt expiry lock")

	select {
	case err := <-result:
		if !errors.Is(err, ErrPairingAttemptUnavailable) {
			t.Fatalf("LockOpenPairingAttempt(expired after wait) error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("LockOpenPairingAttempt(expired after wait) did not return")
	}
	rollbackSessionIntegrationTx(t, waitingTx, "pairing attempt expiry waiter")
}

func waitForPairingAttemptRowLock(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	backendPID uint32,
) {
	t.Helper()
	waitCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	for {
		var waiting bool
		if err := pool.QueryRow(waitCtx, `
			SELECT EXISTS (
				SELECT 1 FROM pg_stat_activity
				WHERE pid = $1 AND wait_event_type = 'Lock'
			)`, backendPID).Scan(&waiting); err != nil {
			t.Fatalf("inspect pairing-attempt lock waiter: %v", err)
		}
		if waiting {
			return
		}
		select {
		case <-waitCtx.Done():
			t.Fatalf("pairing-attempt lock waiter was not observed: %v", waitCtx.Err())
		case <-ticker.C:
		}
	}
}

func newPairingAttemptRepositoryFixture(t *testing.T, createdAt time.Time) PairingAttempt {
	t.Helper()
	id, err := ids.New()
	if err != nil {
		t.Fatalf("ids.New(pairing attempt) error = %v", err)
	}
	spec := validPairingAttemptSpec(t)
	spec.ID = id.String()
	spec.CreatedAt = createdAt
	spec.ExpiresAt = createdAt.Add(maxPairingAttemptLifetime)
	hash, err := HashPairingBootstrapCredential(spec.ID, validPairingBootstrapCredential())
	if err != nil {
		t.Fatalf("HashPairingBootstrapCredential(fixture) error = %v", err)
	}
	spec.BootstrapCredentialHash = hash[:]
	attempt, err := NewPairingAttempt(spec)
	if err != nil {
		t.Fatalf("NewPairingAttempt(fixture) error = %v", err)
	}
	return attempt
}

func assertPairingAttemptFailureCount(
	t *testing.T,
	ctx context.Context,
	reader sessionRowReader,
	id string,
	want int,
) {
	t.Helper()
	var got int
	if err := reader.QueryRow(ctx, `
		SELECT failed_attempt_count FROM session.pairing_attempts WHERE id = $1`, id,
	).Scan(&got); err != nil {
		t.Fatalf("read pairing-attempt failure count: %v", err)
	}
	if got != want {
		t.Fatalf("pairing-attempt failure count = %d, want %d", got, want)
	}
}

func assertPairingAttemptClaimed(
	t *testing.T,
	ctx context.Context,
	reader sessionRowReader,
	id string,
	want PairingAttemptClaim,
	wantClaimedAt time.Time,
) {
	t.Helper()
	var state, ownerID, deviceID, displayName, platform, fingerprint string
	var publicKey []byte
	var claimedAt, updatedAt time.Time
	var hasNoConfirmation bool
	if err := reader.QueryRow(ctx, `
		SELECT state, claimed_user_id, device_id, device_display_name, device_platform,
		       device_static_public_key, device_key_fingerprint, claimed_at, updated_at,
		       mobile_channel_binding IS NULL AND mobile_confirmed_at IS NULL
		         AND agent_channel_binding IS NULL AND agent_confirmed_at IS NULL
		         AND consumed_at IS NULL
		FROM session.pairing_attempts WHERE id = $1`, id,
	).Scan(
		&state, &ownerID, &deviceID, &displayName, &platform, &publicKey, &fingerprint,
		&claimedAt, &updatedAt, &hasNoConfirmation,
	); err != nil {
		t.Fatalf("read claimed pairing attempt: %v", err)
	}
	wantPublicKey, err := want.devicePublicKey.Bytes()
	if err != nil {
		t.Fatalf("read wanted device public key: %v", err)
	}
	wantFingerprint, err := want.deviceFingerprint.String()
	if err != nil {
		t.Fatalf("read wanted device fingerprint: %v", err)
	}
	if state != string(pairingAttemptStateClaimed) || ownerID != want.ownerID.String() ||
		deviceID != want.deviceID.String() || displayName != want.deviceDisplayName ||
		platform != string(want.devicePlatform) || !bytes.Equal(publicKey, wantPublicKey) ||
		fingerprint != wantFingerprint || !claimedAt.Equal(wantClaimedAt) ||
		!updatedAt.Equal(wantClaimedAt) || !hasNoConfirmation {
		t.Fatal("stored pairing attempt did not preserve the exact owner-bound claim")
	}
}

func assertStoredPairingAttempt(
	t *testing.T,
	ctx context.Context,
	reader sessionRowReader,
	want PairingAttempt,
) {
	t.Helper()
	var publicKey, bootstrapHash []byte
	var fingerprint, displayName, version, relayRegion, state string
	var protocolVersion, failedAttemptCount int
	var expiresAt, createdAt, updatedAt time.Time
	var hasNoClaimOrConfirmation bool
	if err := reader.QueryRow(ctx, `
		SELECT agent_static_public_key, agent_key_fingerprint, agent_display_name,
		       agent_version, protocol_version, relay_region, bootstrap_credential_hash,
		       expires_at, failed_attempt_count, state, created_at, updated_at,
		       claimed_user_id IS NULL AND device_id IS NULL
		         AND device_display_name IS NULL AND device_platform IS NULL
		         AND device_static_public_key IS NULL AND device_key_fingerprint IS NULL
		         AND claimed_at IS NULL AND mobile_channel_binding IS NULL
		         AND mobile_confirmed_at IS NULL AND agent_channel_binding IS NULL
		         AND agent_confirmed_at IS NULL AND consumed_at IS NULL
		FROM session.pairing_attempts WHERE id = $1`, want.id.String()).Scan(
		&publicKey, &fingerprint, &displayName, &version, &protocolVersion, &relayRegion,
		&bootstrapHash, &expiresAt, &failedAttemptCount, &state, &createdAt, &updatedAt,
		&hasNoClaimOrConfirmation,
	); err != nil {
		t.Fatalf("read stored pairing attempt: %v", err)
	}
	wantPublicKey, err := want.agentPublicKey.Bytes()
	if err != nil {
		t.Fatalf("read wanted pairing public key: %v", err)
	}
	wantFingerprint, err := want.agentFingerprint.String()
	if err != nil {
		t.Fatalf("read wanted pairing fingerprint: %v", err)
	}
	if !bytes.Equal(publicKey, wantPublicKey) || fingerprint != wantFingerprint ||
		displayName != want.agentDisplayName || version != want.agentVersion ||
		protocolVersion != want.protocolVersion || relayRegion != want.relayRegion ||
		!bytes.Equal(bootstrapHash, want.bootstrapCredentialHash[:]) || !expiresAt.Equal(want.expiresAt) ||
		failedAttemptCount != 0 || state != string(pairingAttemptStateOpen) ||
		!createdAt.Equal(want.createdAt) || !updatedAt.Equal(want.updatedAt) || !hasNoClaimOrConfirmation {
		t.Fatal("stored pairing attempt did not preserve the bounded open-attempt contract")
	}
}

func assertOpenPairingAttemptUnavailable(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	repository *Repository,
	id string,
	checkedAt time.Time,
) {
	t.Helper()
	repository.now = func() time.Time { return checkedAt }
	tx := beginSessionIntegrationTx(t, ctx, pool, "unavailable pairing attempt")
	if _, err := repository.LockOpenPairingAttempt(
		ctx, tx, id,
	); !errors.Is(err, ErrPairingAttemptUnavailable) {
		t.Fatalf("LockOpenPairingAttempt(unavailable) error = %v", err)
	}
	rollbackSessionIntegrationTx(t, tx, "unavailable pairing attempt")
}

func assertPairingAttemptMissing(t *testing.T, ctx context.Context, reader sessionRowReader, id string) {
	t.Helper()
	var storedID string
	err := reader.QueryRow(ctx, `SELECT id FROM session.pairing_attempts WHERE id = $1`, id).Scan(&storedID)
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("pairing attempt %s query error = %v, want no rows", id, err)
	}
}

func deletePairingAttemptFixture(t *testing.T, pool *pgxpool.Pool, id string) {
	t.Helper()
	cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cleanupCancel()
	if _, err := pool.Exec(cleanupCtx, `DELETE FROM session.pairing_attempts WHERE id = $1`, id); err != nil {
		t.Fatalf("delete pairing attempt fixture: %v", err)
	}
}

type commitAcknowledgementLostStarter struct {
	pool *pgxpool.Pool
	err  error
}

func (starter *commitAcknowledgementLostStarter) Begin(ctx context.Context) (pgx.Tx, error) {
	tx, err := starter.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return &commitAcknowledgementLostTx{Tx: tx, err: starter.err}, nil
}

type commitAcknowledgementLostTx struct {
	pgx.Tx
	err error
}

func (tx *commitAcknowledgementLostTx) Commit(ctx context.Context) error {
	if err := tx.Tx.Commit(ctx); err != nil {
		return err
	}
	return tx.err
}
