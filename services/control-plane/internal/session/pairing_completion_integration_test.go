package session

import (
	"bytes"
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/hgunduzoglu/coderoam/packages/go/cryptox"
	"github.com/hgunduzoglu/coderoam/packages/go/ids"
	"github.com/hgunduzoglu/coderoam/packages/go/postgresx"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/auth"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/device"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/workspace"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pairingCompletionIntegrationFixture struct {
	attempt            PairingAttempt
	claim              PairingAttemptClaim
	mobileConfirmation MobilePairingConfirmation
	agentConfirmation  AgentPairingConfirmation
	clock              *time.Time
}

type pairingCompletionIntegrationResult struct {
	completion PairingCompletion
	err        error
}

type observingPairingCompletionStarter struct {
	pool    *pgxpool.Pool
	started chan<- uint32
}

func (starter *observingPairingCompletionStarter) Begin(ctx context.Context) (pgx.Tx, error) {
	tx, err := starter.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	starter.started <- tx.Conn().PgConn().PID()
	return tx, nil
}

type blockingPairingDeviceRegistrar struct {
	repository *device.Repository
	entered    chan<- struct{}
	release    <-chan struct{}
}

type consumeFailingPairingCompletionStore struct {
	repository *Repository
	err        error
}

func (store *consumeFailingPairingCompletionStore) lockPairingCompletion(
	ctx context.Context,
	tx pgx.Tx,
	encodedID string,
	credential []byte,
) (pairingCompletionCandidate, error) {
	return store.repository.lockPairingCompletion(ctx, tx, encodedID, credential)
}

func (store *consumeFailingPairingCompletionStore) consumePairingAttempt(
	context.Context,
	pgx.Tx,
	pairingCompletionCandidate,
) error {
	return store.err
}

func (registrar *blockingPairingDeviceRegistrar) RegisterPaired(
	ctx context.Context,
	tx pgx.Tx,
	ownerID auth.UserID,
	deviceID string,
	name string,
	platform device.Platform,
	publicKey cryptox.X25519PublicKey,
	pairedAt time.Time,
) error {
	if err := registrar.repository.RegisterPaired(
		ctx, tx, ownerID, deviceID, name, platform, publicKey, pairedAt,
	); err != nil {
		return err
	}
	registrar.entered <- struct{}{}
	select {
	case <-registrar.release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func TestPairingCompletionIntegration(t *testing.T) {
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
	applySessionServiceIntegrationMigrations(t, ctx, pool)

	t.Run("atomic commit and exact reconciliation", func(t *testing.T) {
		fixture, attempts, devices, agents := newConfirmedPairingIntegrationFixture(
			t, ctx, pool, time.Date(2026, time.August, 19, 10, 0, 0, 0, time.UTC), true,
		)
		firstPairedAt := fixture.attempt.createdAt.Add(-time.Hour)
		registrationTx := beginSessionIntegrationTx(t, ctx, pool, "existing paired device")
		if err := devices.RegisterPaired(
			ctx, registrationTx, fixture.claim.ownerID, fixture.claim.deviceID.String(),
			fixture.claim.deviceDisplayName, fixture.claim.devicePlatform,
			fixture.claim.devicePublicKey, firstPairedAt,
		); err != nil {
			t.Fatalf("RegisterPaired(existing device fixture) error = %v", err)
		}
		if err := registrationTx.Commit(ctx); err != nil {
			t.Fatalf("commit existing paired device fixture: %v", err)
		}

		*fixture.clock = fixture.attempt.createdAt.Add(4 * time.Minute)
		ackErr := errors.New("completion commit acknowledgement lost")
		ambiguous, err := newPairingCompletionService(
			&commitAcknowledgementLostStarter{pool: pool, err: ackErr}, attempts, devices, agents,
		)
		if err != nil {
			t.Fatalf("newPairingCompletionService(ambiguous) error = %v", err)
		}
		if completed, completionErr := ambiguous.complete(
			ctx, fixture.attempt.id.String(), validPairingBootstrapCredential(),
		); completed != (PairingCompletion{}) ||
			!errors.Is(completionErr, ErrPairingAttemptCommitOutcomeUnknown) ||
			!errors.Is(completionErr, ackErr) {
			t.Fatalf("complete(ambiguous) = (%v, %v)", completed, completionErr)
		}

		consumedAt := assertAtomicPairingCompletion(
			t, ctx, pool, fixture, firstPairedAt,
		)
		*fixture.clock = fixture.attempt.expiresAt.Add(time.Hour)
		service, err := newPairingCompletionService(pool, attempts, devices, agents)
		if err != nil {
			t.Fatalf("newPairingCompletionService(reconcile) error = %v", err)
		}
		wrongCredential := validPairingBootstrapCredential()
		wrongCredential[0] ^= 0xff
		if completed, completionErr := service.complete(
			ctx, fixture.attempt.id.String(), wrongCredential,
		); completed != (PairingCompletion{}) ||
			!errors.Is(completionErr, ErrPairingAttemptUnavailable) {
			t.Fatalf("complete(consumed wrong credential) = (%v, %v)", completed, completionErr)
		}
		assertPairingCompletionCounts(t, ctx, pool, fixture, "consumed", true, 1, 1)
		completed, err := service.complete(
			ctx, fixture.attempt.id.String(), validPairingBootstrapCredential(),
		)
		if err != nil {
			t.Fatalf("complete(reconcile after expiry) error = %v", err)
		}
		if completed.PairingID != fixture.attempt.id.String() ||
			completed.DeviceID != fixture.claim.deviceID.String() ||
			completed.AgentID != fixture.attempt.agentID.String() ||
			!completed.ConsumedAt.Equal(consumedAt) {
			t.Fatal("completion reconciliation did not preserve the committed result")
		}

		confirmationService, err := newPairingAttemptService(pool, attempts)
		if err != nil {
			t.Fatalf("newPairingAttemptService(consumed retry) error = %v", err)
		}
		if err := confirmationService.confirmMobile(
			ctx, fixture.attempt.id.String(), fixture.mobileConfirmation,
		); err != nil {
			t.Fatalf("confirmMobile(consumed exact retry) error = %v", err)
		}
		if err := confirmationService.confirmAgent(
			ctx, fixture.attempt.id.String(), validPairingBootstrapCredential(),
			fixture.agentConfirmation,
		); err != nil {
			t.Fatalf("confirmAgent(consumed exact retry) error = %v", err)
		}
	})

	t.Run("missing peer confirmation creates no registration", func(t *testing.T) {
		fixture, attempts, devices, agents := newConfirmedPairingIntegrationFixture(
			t, ctx, pool, time.Date(2026, time.August, 19, 11, 0, 0, 0, time.UTC), false,
		)
		*fixture.clock = fixture.attempt.createdAt.Add(3 * time.Minute)
		service, err := newPairingCompletionService(pool, attempts, devices, agents)
		if err != nil {
			t.Fatalf("newPairingCompletionService(pending) error = %v", err)
		}
		if completed, completionErr := service.complete(
			ctx, fixture.attempt.id.String(), validPairingBootstrapCredential(),
		); completed != (PairingCompletion{}) ||
			!errors.Is(completionErr, ErrPairingAttemptUnavailable) {
			t.Fatalf("complete(pending) = (%v, %v)", completed, completionErr)
		}
		assertPairingCompletionCounts(t, ctx, pool, fixture, "confirming", false, 0, 0)
	})

	t.Run("wrong bootstrap credential creates no registration", func(t *testing.T) {
		fixture, attempts, devices, agents := newConfirmedPairingIntegrationFixture(
			t, ctx, pool, time.Date(2026, time.August, 19, 11, 30, 0, 0, time.UTC), true,
		)
		*fixture.clock = fixture.attempt.createdAt.Add(4 * time.Minute)
		service, err := newPairingCompletionService(pool, attempts, devices, agents)
		if err != nil {
			t.Fatalf("newPairingCompletionService(wrong credential) error = %v", err)
		}
		wrongCredential := validPairingBootstrapCredential()
		wrongCredential[0] ^= 0xff
		if completed, completionErr := service.complete(
			ctx, fixture.attempt.id.String(), wrongCredential,
		); completed != (PairingCompletion{}) ||
			!errors.Is(completionErr, ErrPairingAttemptUnavailable) {
			t.Fatalf("complete(wrong credential) = (%v, %v)", completed, completionErr)
		}
		assertPairingCompletionCounts(t, ctx, pool, fixture, "confirming", false, 0, 0)
	})

	t.Run("revoked device prevents agent registration and consumption", func(t *testing.T) {
		fixture, attempts, devices, agents := newConfirmedPairingIntegrationFixture(
			t, ctx, pool, time.Date(2026, time.August, 19, 11, 40, 0, 0, time.UTC), true,
		)
		*fixture.clock = fixture.attempt.createdAt.Add(4 * time.Minute)
		seedTx := beginSessionIntegrationTx(t, ctx, pool, "revoked device fixture")
		if err := devices.RegisterPaired(
			ctx, seedTx, fixture.claim.ownerID, fixture.claim.deviceID.String(),
			fixture.claim.deviceDisplayName, fixture.claim.devicePlatform,
			fixture.claim.devicePublicKey, *fixture.clock,
		); err != nil {
			t.Fatalf("RegisterPaired(revoked device fixture) error = %v", err)
		}
		if _, err := seedTx.Exec(
			ctx, `UPDATE device.devices SET revoked_at = $1 WHERE id = $2`,
			fixture.clock.Add(time.Minute), fixture.claim.deviceID.String(),
		); err != nil {
			t.Fatalf("revoke device fixture: %v", err)
		}
		if err := seedTx.Commit(ctx); err != nil {
			t.Fatalf("commit revoked device fixture: %v", err)
		}

		service, err := newPairingCompletionService(pool, attempts, devices, agents)
		if err != nil {
			t.Fatalf("newPairingCompletionService(revoked device) error = %v", err)
		}
		if completed, completionErr := service.complete(
			ctx, fixture.attempt.id.String(), validPairingBootstrapCredential(),
		); completed != (PairingCompletion{}) ||
			!errors.Is(completionErr, ErrPairingAttemptUnavailable) {
			t.Fatalf("complete(revoked device) = (%v, %v)", completed, completionErr)
		}
		assertPairingCompletionCounts(t, ctx, pool, fixture, "confirming", false, 1, 0)
	})

	t.Run("revoked agent rolls back device and consumption", func(t *testing.T) {
		fixture, attempts, devices, agents := newConfirmedPairingIntegrationFixture(
			t, ctx, pool, time.Date(2026, time.August, 19, 11, 50, 0, 0, time.UTC), true,
		)
		*fixture.clock = fixture.attempt.createdAt.Add(4 * time.Minute)
		seedTx := beginSessionIntegrationTx(t, ctx, pool, "revoked agent fixture")
		if err := agents.RegisterPairedAgent(
			ctx, seedTx, fixture.claim.ownerID, fixture.attempt.agentID.String(),
			fixture.attempt.agentDisplayName, fixture.attempt.agentPublicKey,
			fixture.attempt.agentVersion, *fixture.clock,
		); err != nil {
			t.Fatalf("RegisterPairedAgent(revoked agent fixture) error = %v", err)
		}
		if _, err := seedTx.Exec(
			ctx, `UPDATE workspace.agents SET revoked_at = $1 WHERE id = $2`,
			fixture.clock.Add(time.Minute), fixture.attempt.agentID.String(),
		); err != nil {
			t.Fatalf("revoke agent fixture: %v", err)
		}
		if err := seedTx.Commit(ctx); err != nil {
			t.Fatalf("commit revoked agent fixture: %v", err)
		}

		service, err := newPairingCompletionService(pool, attempts, devices, agents)
		if err != nil {
			t.Fatalf("newPairingCompletionService(revoked agent) error = %v", err)
		}
		if completed, completionErr := service.complete(
			ctx, fixture.attempt.id.String(), validPairingBootstrapCredential(),
		); completed != (PairingCompletion{}) ||
			!errors.Is(completionErr, ErrPairingAttemptUnavailable) {
			t.Fatalf("complete(revoked agent) = (%v, %v)", completed, completionErr)
		}
		assertPairingCompletionCounts(t, ctx, pool, fixture, "confirming", false, 0, 1)
	})

	t.Run("consume failure rolls back both registrations", func(t *testing.T) {
		fixture, attempts, devices, agents := newConfirmedPairingIntegrationFixture(
			t, ctx, pool, time.Date(2026, time.August, 19, 11, 55, 0, 0, time.UTC), true,
		)
		*fixture.clock = fixture.attempt.createdAt.Add(4 * time.Minute)
		consumeErr := errors.New("consume failed after endpoint registration")
		store := &consumeFailingPairingCompletionStore{repository: attempts, err: consumeErr}
		service, err := newPairingCompletionService(pool, store, devices, agents)
		if err != nil {
			t.Fatalf("newPairingCompletionService(consume failure) error = %v", err)
		}
		if completed, completionErr := service.complete(
			ctx, fixture.attempt.id.String(), validPairingBootstrapCredential(),
		); completed != (PairingCompletion{}) ||
			!errors.Is(completionErr, ErrPairingAttemptPersistenceUnavailable) ||
			!errors.Is(completionErr, consumeErr) {
			t.Fatalf("complete(consume failure) = (%v, %v)", completed, completionErr)
		}
		assertPairingCompletionCounts(t, ctx, pool, fixture, "confirming", false, 0, 0)
	})

	t.Run("agent collision rolls back device and consumption", func(t *testing.T) {
		fixture, attempts, devices, agents := newConfirmedPairingIntegrationFixture(
			t, ctx, pool, time.Date(2026, time.August, 19, 12, 0, 0, 0, time.UTC), true,
		)
		foreignOwner, err := auth.ParseUserID("f123456789abcdef0123456789abcdef")
		if err != nil {
			t.Fatalf("ParseUserID(foreign owner) error = %v", err)
		}
		foreignKey, err := cryptox.ParseX25519PublicKey(bytes.Repeat([]byte{0x71}, 32))
		if err != nil {
			t.Fatalf("ParseX25519PublicKey(foreign agent) error = %v", err)
		}
		collisionTx := beginSessionIntegrationTx(t, ctx, pool, "agent collision fixture")
		if err := agents.RegisterPairedAgent(
			ctx, collisionTx, foreignOwner, fixture.attempt.agentID.String(),
			"Foreign agent", foreignKey, "9.9.9", fixture.attempt.createdAt,
		); err != nil {
			t.Fatalf("RegisterPairedAgent(collision fixture) error = %v", err)
		}
		if err := collisionTx.Commit(ctx); err != nil {
			t.Fatalf("commit agent collision fixture: %v", err)
		}

		*fixture.clock = fixture.attempt.createdAt.Add(4 * time.Minute)
		service, err := newPairingCompletionService(pool, attempts, devices, agents)
		if err != nil {
			t.Fatalf("newPairingCompletionService(collision) error = %v", err)
		}
		if completed, completionErr := service.complete(
			ctx, fixture.attempt.id.String(), validPairingBootstrapCredential(),
		); completed != (PairingCompletion{}) ||
			!errors.Is(completionErr, ErrPairingAttemptUnavailable) {
			t.Fatalf("complete(agent collision) = (%v, %v)", completed, completionErr)
		}
		assertPairingCompletionCounts(t, ctx, pool, fixture, "confirming", false, 0, 1)
	})

	t.Run("concurrent exact completion is stable", func(t *testing.T) {
		fixture, attempts, devices, agents := newConfirmedPairingIntegrationFixture(
			t, ctx, pool, time.Date(2026, time.August, 19, 13, 0, 0, 0, time.UTC), true,
		)
		*fixture.clock = fixture.attempt.createdAt.Add(4 * time.Minute)
		entered := make(chan struct{}, 1)
		release := make(chan struct{})
		var releaseOnce sync.Once
		releaseCompletion := func() { releaseOnce.Do(func() { close(release) }) }
		t.Cleanup(releaseCompletion)
		blockingDevices := &blockingPairingDeviceRegistrar{
			repository: devices, entered: entered, release: release,
		}
		firstService, err := newPairingCompletionService(
			pool, attempts, blockingDevices, agents,
		)
		if err != nil {
			t.Fatalf("newPairingCompletionService(first concurrent) error = %v", err)
		}
		firstResult := make(chan pairingCompletionIntegrationResult, 1)
		go func() {
			completion, completionErr := firstService.complete(
				ctx, fixture.attempt.id.String(), validPairingBootstrapCredential(),
			)
			firstResult <- pairingCompletionIntegrationResult{completion: completion, err: completionErr}
		}()
		select {
		case <-entered:
		case <-ctx.Done():
			t.Fatalf("first concurrent completion did not reach registration: %v", ctx.Err())
		}

		started := make(chan uint32, 1)
		secondService, err := newPairingCompletionService(
			&observingPairingCompletionStarter{pool: pool, started: started}, attempts, devices, agents,
		)
		if err != nil {
			t.Fatalf("newPairingCompletionService(second concurrent) error = %v", err)
		}
		secondResult := make(chan pairingCompletionIntegrationResult, 1)
		go func() {
			completion, completionErr := secondService.complete(
				ctx, fixture.attempt.id.String(), validPairingBootstrapCredential(),
			)
			secondResult <- pairingCompletionIntegrationResult{completion: completion, err: completionErr}
		}()
		var secondPID uint32
		select {
		case secondPID = <-started:
		case <-ctx.Done():
			t.Fatalf("second concurrent completion did not begin: %v", ctx.Err())
		}
		waitForPairingAttemptRowLock(t, ctx, pool, secondPID)
		releaseCompletion()

		first := <-firstResult
		second := <-secondResult
		if first.err != nil || second.err != nil {
			t.Fatalf("concurrent completion errors = (%v, %v)", first.err, second.err)
		}
		if first.completion != second.completion {
			t.Fatalf("concurrent completions differ = (%v, %v)", first.completion, second.completion)
		}
		assertPairingCompletionCounts(t, ctx, pool, fixture, "consumed", true, 1, 1)
	})
}

func newConfirmedPairingIntegrationFixture(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	createdAt time.Time,
	confirmAgent bool,
) (pairingCompletionIntegrationFixture, *Repository, *device.Repository, *workspace.Repository) {
	t.Helper()
	clock := createdAt
	attempts := NewRepository()
	attempts.now = func() time.Time { return clock }
	devices, err := device.NewRepository(pool, func() time.Time { return clock })
	if err != nil {
		t.Fatalf("device.NewRepository() error = %v", err)
	}
	agents, err := workspace.NewRepository(pool, func() time.Time { return clock })
	if err != nil {
		t.Fatalf("workspace.NewRepository() error = %v", err)
	}

	attempt := newPairingAttemptRepositoryFixture(t, createdAt)
	agentID, err := ids.New()
	if err != nil {
		t.Fatalf("ids.New(agent) error = %v", err)
	}
	attempt.agentID = agentID
	claimSpec := validPairingAttemptClaimSpec(t)
	deviceID, err := ids.New()
	if err != nil {
		t.Fatalf("ids.New(device) error = %v", err)
	}
	claimSpec.DeviceID = deviceID.String()
	claim, err := NewPairingAttemptClaim(claimSpec)
	if err != nil {
		t.Fatalf("NewPairingAttemptClaim() error = %v", err)
	}
	mobileConfirmation, err := NewMobilePairingConfirmation(validMobilePairingConfirmationSpec(t))
	if err != nil {
		t.Fatalf("NewMobilePairingConfirmation() error = %v", err)
	}
	agentConfirmation, err := NewAgentPairingConfirmation(validAgentPairingConfirmationSpec(t))
	if err != nil {
		t.Fatalf("NewAgentPairingConfirmation() error = %v", err)
	}

	deletePairingCompletionFixture(t, ctx, pool, attempt, claim)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		deletePairingCompletionFixture(t, cleanupCtx, pool, attempt, claim)
	})
	createTx := beginSessionIntegrationTx(t, ctx, pool, "pairing completion create")
	if err := attempts.CreatePairingAttempt(ctx, createTx, attempt); err != nil {
		t.Fatalf("CreatePairingAttempt(completion) error = %v", err)
	}
	if err := createTx.Commit(ctx); err != nil {
		t.Fatalf("commit pairing completion attempt: %v", err)
	}

	confirmationService, err := newPairingAttemptService(pool, attempts)
	if err != nil {
		t.Fatalf("newPairingAttemptService(completion) error = %v", err)
	}
	clock = createdAt.Add(time.Minute)
	if err := confirmationService.claim(ctx, attempt.id.String(), claim); err != nil {
		t.Fatalf("claim(completion) error = %v", err)
	}
	clock = createdAt.Add(2 * time.Minute)
	if err := confirmationService.confirmMobile(
		ctx, attempt.id.String(), mobileConfirmation,
	); err != nil {
		t.Fatalf("confirmMobile(completion) error = %v", err)
	}
	if confirmAgent {
		clock = createdAt.Add(3 * time.Minute)
		if err := confirmationService.confirmAgent(
			ctx, attempt.id.String(), validPairingBootstrapCredential(), agentConfirmation,
		); err != nil {
			t.Fatalf("confirmAgent(completion) error = %v", err)
		}
	}
	return pairingCompletionIntegrationFixture{
		attempt: attempt, claim: claim, mobileConfirmation: mobileConfirmation,
		agentConfirmation: agentConfirmation, clock: &clock,
	}, attempts, devices, agents
}

func assertAtomicPairingCompletion(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	fixture pairingCompletionIntegrationFixture,
	wantDevicePairedAt time.Time,
) time.Time {
	t.Helper()
	assertPairingCompletionCounts(t, ctx, pool, fixture, "consumed", true, 1, 1)
	var consumedAt, devicePairedAt, agentCreatedAt time.Time
	if err := pool.QueryRow(ctx, `
		SELECT p.consumed_at, d.paired_at, a.created_at
		FROM session.pairing_attempts AS p
		JOIN device.devices AS d ON d.id = p.device_id
		JOIN workspace.agents AS a ON a.id = p.agent_id
		WHERE p.id = $1`, fixture.attempt.id.String()).Scan(
		&consumedAt, &devicePairedAt, &agentCreatedAt,
	); err != nil {
		t.Fatalf("read atomic pairing completion: %v", err)
	}
	if !devicePairedAt.Equal(wantDevicePairedAt) || !agentCreatedAt.Equal(consumedAt) {
		t.Fatal("atomic completion replaced the first device time or changed the agent registration time")
	}
	return consumedAt
}

func assertPairingCompletionCounts(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	fixture pairingCompletionIntegrationFixture,
	wantState string,
	wantConsumed bool,
	wantDeviceCount int,
	wantAgentCount int,
) {
	t.Helper()
	var state string
	var consumed bool
	var deviceCount, agentCount int
	if err := pool.QueryRow(ctx, `
		SELECT state, consumed_at IS NOT NULL
		FROM session.pairing_attempts WHERE id = $1`, fixture.attempt.id.String(),
	).Scan(&state, &consumed); err != nil {
		t.Fatalf("read pairing completion state: %v", err)
	}
	if err := pool.QueryRow(
		ctx, `SELECT count(*) FROM device.devices WHERE id = $1`, fixture.claim.deviceID.String(),
	).Scan(&deviceCount); err != nil {
		t.Fatalf("count pairing completion device: %v", err)
	}
	if err := pool.QueryRow(
		ctx, `SELECT count(*) FROM workspace.agents WHERE id = $1`, fixture.attempt.agentID.String(),
	).Scan(&agentCount); err != nil {
		t.Fatalf("count pairing completion agent: %v", err)
	}
	if state != wantState || consumed != wantConsumed ||
		deviceCount != wantDeviceCount || agentCount != wantAgentCount {
		t.Fatalf(
			"completion state/counts = %s/%t/%d/%d, want %s/%t/%d/%d",
			state, consumed, deviceCount, agentCount,
			wantState, wantConsumed, wantDeviceCount, wantAgentCount,
		)
	}
}

func deletePairingCompletionFixture(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	attempt PairingAttempt,
	claim PairingAttemptClaim,
) {
	t.Helper()
	if _, err := pool.Exec(ctx, `DELETE FROM session.pairing_attempts WHERE id = $1`, attempt.id.String()); err != nil {
		t.Fatalf("delete pairing completion attempt: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM device.devices WHERE id = $1`, claim.deviceID.String()); err != nil {
		t.Fatalf("delete pairing completion device: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM workspace.agents WHERE id = $1`, attempt.agentID.String()); err != nil {
		t.Fatalf("delete pairing completion agent: %v", err)
	}
}
