package session

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/hgunduzoglu/coderoam/packages/go/ids"
	"github.com/hgunduzoglu/coderoam/packages/go/postgresx"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/auth"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/device"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/workspace"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type sessionServiceFixture struct {
	deviceID       string
	agentID        string
	otherAgentID   string
	environmentID  string
	projectID      string
	devicePairedAt time.Time
	agentCreatedAt time.Time
}

type sessionServiceStartResult struct {
	metadata Session
	err      error
}

type blockingSessionCreator struct {
	repository *Repository
	inserted   chan struct{}
	release    chan struct{}
}

type ambiguousSessionCommitStarter struct {
	pool *pgxpool.Pool
}

func (starter *ambiguousSessionCommitStarter) Begin(ctx context.Context) (pgx.Tx, error) {
	tx, err := starter.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return &ambiguousSessionCommitTx{Tx: tx}, nil
}

type ambiguousSessionCommitTx struct {
	pgx.Tx
}

func (tx *ambiguousSessionCommitTx) Commit(ctx context.Context) error {
	if err := tx.Tx.Commit(ctx); err != nil {
		return err
	}
	return errors.New("simulated lost commit acknowledgement")
}

func (creator *blockingSessionCreator) CreateOrGet(
	ctx context.Context,
	tx pgx.Tx,
	metadata Session,
) (Session, error) {
	persisted, err := creator.repository.CreateOrGet(ctx, tx, metadata)
	if err != nil {
		return Session{}, err
	}
	close(creator.inserted)
	select {
	case <-creator.release:
		return persisted, nil
	case <-ctx.Done():
		return Session{}, ctx.Err()
	}
}

func TestServiceStartIntegration(t *testing.T) {
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

	owner := newSessionTestActor(t, newSessionServiceIntegrationID(t))
	foreign := newSessionTestActor(t, newSessionServiceIntegrationID(t))
	ownerID, _ := owner.UserID()
	checkedAt := time.Date(2026, time.July, 19, 17, 0, 0, 0, time.UTC)
	fixture := insertSessionServiceFixture(t, ctx, pool, ownerID.String(), checkedAt)
	clock := func() time.Time { return checkedAt }
	deviceRepository, err := device.NewRepository(pool, clock)
	if err != nil {
		t.Fatalf("device.NewRepository() error = %v", err)
	}
	workspaceRepository, err := workspace.NewRepository(pool, clock)
	if err != nil {
		t.Fatalf("workspace.NewRepository() error = %v", err)
	}
	sessionRepository := NewRepository()

	committedID := newSessionServiceIntegrationID(t)
	assertSessionMissing(t, ctx, pool, committedID)
	deleteSessionIntegrationFixture(t, pool, committedID)
	service := newSessionIntegrationService(
		t, pool, deviceRepository, workspaceRepository, sessionRepository, clock,
	)
	committed, err := service.Start(
		ctx, owner, committedID, fixture.deviceID, fixture.agentID, fixture.projectID,
	)
	if err != nil {
		t.Fatalf("Start(owner resources) error = %v", err)
	}
	if committed.id.String() != committedID || !committed.OwnedBy(owner) {
		t.Fatal("Start(owner resources) returned unexpected session metadata")
	}
	assertStoredSession(t, ctx, pool, committed)
	ambiguousID := newSessionServiceIntegrationID(t)
	assertSessionMissing(t, ctx, pool, ambiguousID)
	deleteSessionIntegrationFixture(t, pool, ambiguousID)
	ambiguousService := newSessionIntegrationService(
		t, &ambiguousSessionCommitStarter{pool: pool}, deviceRepository,
		workspaceRepository, sessionRepository, clock,
	)
	if started, err := ambiguousService.Start(
		ctx, owner, ambiguousID, fixture.deviceID, fixture.agentID, fixture.projectID,
	); !errors.Is(err, ErrSessionCommitOutcomeUnknown) || started != (Session{}) {
		t.Fatalf("Start(ambiguous commit) = (%v, %v), want zero session and outcome unknown", started, err)
	}
	assertSessionRowCount(t, ctx, pool, ambiguousID, 1)
	retryService := newSessionIntegrationService(
		t, pool, deviceRepository, workspaceRepository, sessionRepository, clock,
	)
	reconciled, err := retryService.Start(
		ctx, owner, ambiguousID, fixture.deviceID, fixture.agentID, fixture.projectID,
	)
	if err != nil {
		t.Fatalf("Start(after ambiguous commit) error = %v", err)
	}
	if reconciled.id.String() != ambiguousID || !reconciled.OwnedBy(owner) {
		t.Fatal("Start(after ambiguous commit) did not reconcile the committed session")
	}
	assertSessionRowCount(t, ctx, pool, ambiguousID, 1)

	denials := map[string]struct {
		actor     auth.Actor
		deviceID  string
		agentID   string
		projectID string
	}{
		"foreign owner": {actor: foreign, deviceID: fixture.deviceID, agentID: fixture.agentID, projectID: fixture.projectID},
		"missing device": {actor: owner, deviceID: newSessionServiceIntegrationID(t),
			agentID: fixture.agentID, projectID: fixture.projectID},
		"missing agent": {actor: owner, deviceID: fixture.deviceID,
			agentID: newSessionServiceIntegrationID(t), projectID: fixture.projectID},
		"wrong project agent": {actor: owner, deviceID: fixture.deviceID,
			agentID: fixture.otherAgentID, projectID: fixture.projectID},
	}
	for name, denial := range denials {
		t.Run(name, func(t *testing.T) {
			sessionID := newSessionServiceIntegrationID(t)
			assertSessionMissing(t, ctx, pool, sessionID)
			deleteSessionIntegrationFixture(t, pool, sessionID)
			deniedService := newSessionIntegrationService(
				t, pool, deviceRepository, workspaceRepository, sessionRepository, clock,
			)
			started, err := deniedService.Start(
				ctx, denial.actor, sessionID, denial.deviceID, denial.agentID, denial.projectID,
			)
			if !errors.Is(err, ErrSessionAccessDenied) {
				t.Fatalf("Start() error = %v, want ErrSessionAccessDenied", err)
			}
			if started != (Session{}) {
				t.Fatal("denied Start() returned session metadata")
			}
			assertSessionMissing(t, ctx, pool, sessionID)
		})
	}

	duplicateService := newSessionIntegrationService(
		t, pool, deviceRepository, workspaceRepository, sessionRepository, clock,
	)
	retried, err := duplicateService.Start(
		ctx, owner, committedID, fixture.deviceID, fixture.agentID, fixture.projectID,
	)
	if err != nil {
		t.Fatalf("Start(idempotent retry) error = %v", err)
	}
	if retried.id != committed.id || !retried.startedAt.Equal(committed.startedAt) {
		t.Fatal("Start(idempotent retry) did not return the first committed session")
	}
	assertStoredSession(t, ctx, pool, committed)

	lockedSessionID := newSessionServiceIntegrationID(t)
	assertSessionMissing(t, ctx, pool, lockedSessionID)
	deleteSessionIntegrationFixture(t, pool, lockedSessionID)
	blockingCreator := &blockingSessionCreator{
		repository: sessionRepository,
		inserted:   make(chan struct{}),
		release:    make(chan struct{}),
	}
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(blockingCreator.release) }) }
	t.Cleanup(release)
	lockedService := newSessionIntegrationService(
		t, pool, deviceRepository, workspaceRepository, blockingCreator, clock,
	)
	result := make(chan sessionServiceStartResult, 1)
	go func() {
		metadata, err := lockedService.Start(
			ctx, owner, lockedSessionID, fixture.deviceID, fixture.agentID, fixture.projectID,
		)
		result <- sessionServiceStartResult{metadata: metadata, err: err}
	}()
	select {
	case <-blockingCreator.inserted:
	case <-ctx.Done():
		t.Fatalf("wait for locked session insert: %v", ctx.Err())
	}

	revokeCtx, cancelRevoke := context.WithTimeout(context.Background(), 100*time.Millisecond)
	err = deviceRepository.Revoke(revokeCtx, owner, fixture.deviceID)
	cancelRevoke()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Revoke(device during Start) error = %v, want context.DeadlineExceeded", err)
	}
	assertDeviceNotRevoked(t, ctx, pool, fixture.deviceID)
	revokeCtx, cancelRevoke = context.WithTimeout(context.Background(), 100*time.Millisecond)
	err = workspaceRepository.RevokeAgent(revokeCtx, owner, fixture.agentID)
	cancelRevoke()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("RevokeAgent(during Start) error = %v, want context.DeadlineExceeded", err)
	}
	assertAgentNotRevoked(t, ctx, pool, fixture.agentID)
	release()
	select {
	case startResult := <-result:
		if startResult.err != nil {
			t.Fatalf("locked Start() error = %v", startResult.err)
		}
		if startResult.metadata.id.String() != lockedSessionID {
			t.Fatal("locked Start() returned unexpected metadata")
		}
		assertStoredSession(t, ctx, pool, startResult.metadata)
	case <-ctx.Done():
		t.Fatalf("wait for locked Start(): %v", ctx.Err())
	}
	if err := deviceRepository.Revoke(ctx, owner, fixture.deviceID); err != nil {
		t.Fatalf("Revoke(device after Start) error = %v", err)
	}
	assertDeviceRevokedAt(t, ctx, pool, fixture.deviceID, checkedAt)
	if err := workspaceRepository.RevokeAgent(ctx, owner, fixture.agentID); err != nil {
		t.Fatalf("RevokeAgent(after Start) error = %v", err)
	}
	assertAgentRevokedAt(t, ctx, pool, fixture.agentID, checkedAt)
}

func newSessionIntegrationService(
	t *testing.T,
	transactions transactionStarter,
	devices deviceAuthorizer,
	workspaces workspaceAuthorizer,
	sessions sessionCreator,
	now func() time.Time,
) *Service {
	t.Helper()
	service, err := NewService(
		transactions, devices, workspaces, sessions, "eu-central-1", now,
	)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	return service
}

func applySessionServiceIntegrationMigrations(
	t *testing.T,
	ctx context.Context,
	database postgresx.TransactionStarter,
) {
	t.Helper()
	files := []struct {
		scope string
		path  string
	}{
		{scope: "outbox", path: "../outbox/migrations/000001_init.sql"},
		{scope: "device", path: "../device/migrations/000001_init.sql"},
		{scope: "workspace", path: "../workspace/migrations/000001_init.sql"},
		{scope: "session", path: "migrations/000001_init.sql"},
	}
	for _, file := range files {
		sql, err := os.ReadFile(file.path)
		if err != nil {
			t.Fatalf("read %s migration: %v", file.scope, err)
		}
		if err := postgresx.ApplyMigrations(ctx, database, []postgresx.Migration{{
			Scope: file.scope, Version: 1, Name: "init", SQL: string(sql),
		}}); err != nil {
			t.Fatalf("apply %s migration: %v", file.scope, err)
		}
	}
}

func insertSessionServiceFixture(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	ownerID string,
	checkedAt time.Time,
) sessionServiceFixture {
	t.Helper()
	fixture := sessionServiceFixture{
		deviceID:       newSessionServiceIntegrationID(t),
		agentID:        newSessionServiceIntegrationID(t),
		otherAgentID:   newSessionServiceIntegrationID(t),
		environmentID:  newSessionServiceIntegrationID(t),
		projectID:      newSessionServiceIntegrationID(t),
		devicePairedAt: checkedAt.Add(-4 * time.Hour),
		agentCreatedAt: checkedAt.Add(-3 * time.Hour),
	}
	assertSessionServiceFixturesMissing(t, ctx, pool, fixture)
	registerSessionServiceFixtureCleanup(t, pool, fixture)
	if _, err := pool.Exec(ctx, `
		INSERT INTO device.devices (
			id, user_id, name, platform, static_public_key, public_key_fingerprint, paired_at
		) VALUES ($1, $2, 'Owner phone', 'ios', $3, $4, $5)`,
		fixture.deviceID, ownerID, repeatedSessionServiceByte(0x41), "device-"+fixture.deviceID,
		fixture.devicePairedAt,
	); err != nil {
		t.Fatalf("insert session-service device: %v", err)
	}
	for _, agent := range []struct {
		id      string
		keyByte byte
	}{
		{id: fixture.agentID, keyByte: 0x51},
		{id: fixture.otherAgentID, keyByte: 0x52},
	} {
		if _, err := pool.Exec(ctx, `
			INSERT INTO workspace.agents (
				id, user_id, name, static_public_key, public_key_fingerprint, version, created_at
			) VALUES ($1, $2, 'Owner agent', $3, $4, '0.1.0', $5)`,
			agent.id, ownerID, repeatedSessionServiceByte(agent.keyByte), "agent-"+agent.id,
			fixture.agentCreatedAt,
		); err != nil {
			t.Fatalf("insert session-service agent: %v", err)
		}
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO workspace.environments (id, user_id, agent_id, name, provider, created_at)
		VALUES ($1, $2, $3, 'Owner environment', 'linux', $4)`,
		fixture.environmentID, ownerID, fixture.agentID, checkedAt.Add(-2*time.Hour),
	); err != nil {
		t.Fatalf("insert session-service environment: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO workspace.projects (id, user_id, environment_id, name, root_path, created_at)
		VALUES ($1, $2, $3, 'Owner project', '/srv/coderoam/project', $4)`,
		fixture.projectID, ownerID, fixture.environmentID, checkedAt.Add(-time.Hour),
	); err != nil {
		t.Fatalf("insert session-service project: %v", err)
	}
	return fixture
}

func assertSessionServiceFixturesMissing(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	fixture sessionServiceFixture,
) {
	t.Helper()
	lookups := []struct {
		query string
		id    string
	}{
		{query: `SELECT id FROM device.devices WHERE id = $1`, id: fixture.deviceID},
		{query: `SELECT id FROM workspace.agents WHERE id = $1`, id: fixture.agentID},
		{query: `SELECT id FROM workspace.agents WHERE id = $1`, id: fixture.otherAgentID},
		{query: `SELECT id FROM workspace.environments WHERE id = $1`, id: fixture.environmentID},
		{query: `SELECT id FROM workspace.projects WHERE id = $1`, id: fixture.projectID},
		{query: `SELECT id FROM outbox.events WHERE aggregate_type = 'device' AND aggregate_id = $1`, id: fixture.deviceID},
		{query: `SELECT id FROM outbox.events WHERE aggregate_type = 'agent' AND aggregate_id = $1`, id: fixture.agentID},
	}
	for _, lookup := range lookups {
		var storedID string
		if err := pool.QueryRow(ctx, lookup.query, lookup.id).Scan(&storedID); !errors.Is(err, pgx.ErrNoRows) {
			t.Fatalf("fixture %s lookup error = %v, want pgx.ErrNoRows", lookup.id, err)
		}
	}
}

func registerSessionServiceFixtureCleanup(t *testing.T, pool *pgxpool.Pool, fixture sessionServiceFixture) {
	t.Helper()
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		deletes := []struct {
			query string
			id    string
		}{
			{query: `DELETE FROM outbox.events WHERE aggregate_type = 'device' AND aggregate_id = $1`, id: fixture.deviceID},
			{query: `DELETE FROM outbox.events WHERE aggregate_type = 'agent' AND aggregate_id = $1`, id: fixture.agentID},
			{query: `DELETE FROM workspace.projects WHERE id = $1`, id: fixture.projectID},
			{query: `DELETE FROM workspace.environments WHERE id = $1`, id: fixture.environmentID},
			{query: `DELETE FROM workspace.agents WHERE id = $1`, id: fixture.agentID},
			{query: `DELETE FROM workspace.agents WHERE id = $1`, id: fixture.otherAgentID},
			{query: `DELETE FROM device.devices WHERE id = $1`, id: fixture.deviceID},
		}
		for _, deletion := range deletes {
			if _, err := pool.Exec(cleanupCtx, deletion.query, deletion.id); err != nil {
				t.Errorf("delete session-service fixture %s: %v", deletion.id, err)
			}
		}
	})
}

func assertDeviceNotRevoked(t *testing.T, ctx context.Context, pool *pgxpool.Pool, deviceID string) {
	t.Helper()
	var revokedAt *time.Time
	if err := pool.QueryRow(ctx, `SELECT revoked_at FROM device.devices WHERE id = $1`, deviceID).Scan(&revokedAt); err != nil {
		t.Fatalf("read device revocation: %v", err)
	}
	if revokedAt != nil {
		t.Fatalf("device revoked_at = %v, want nil", *revokedAt)
	}
}

func assertDeviceRevokedAt(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	deviceID string,
	want time.Time,
) {
	t.Helper()
	var revokedAt time.Time
	if err := pool.QueryRow(ctx, `SELECT revoked_at FROM device.devices WHERE id = $1`, deviceID).Scan(&revokedAt); err != nil {
		t.Fatalf("read revoked device: %v", err)
	}
	if !revokedAt.Equal(want) {
		t.Fatalf("device revoked_at = %v, want %v", revokedAt, want)
	}
}

func assertAgentNotRevoked(t *testing.T, ctx context.Context, pool *pgxpool.Pool, agentID string) {
	t.Helper()
	var revokedAt *time.Time
	if err := pool.QueryRow(ctx, `SELECT revoked_at FROM workspace.agents WHERE id = $1`, agentID).Scan(&revokedAt); err != nil {
		t.Fatalf("read agent revocation: %v", err)
	}
	if revokedAt != nil {
		t.Fatalf("agent revoked_at = %v, want nil", *revokedAt)
	}
}

func assertAgentRevokedAt(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	agentID string,
	want time.Time,
) {
	t.Helper()
	var revokedAt time.Time
	if err := pool.QueryRow(ctx, `SELECT revoked_at FROM workspace.agents WHERE id = $1`, agentID).Scan(&revokedAt); err != nil {
		t.Fatalf("read revoked agent: %v", err)
	}
	if !revokedAt.Equal(want) {
		t.Fatalf("agent revoked_at = %v, want %v", revokedAt, want)
	}
}

func newSessionServiceIntegrationID(t *testing.T) string {
	t.Helper()
	id, err := ids.New()
	if err != nil {
		t.Fatalf("ids.New() error = %v", err)
	}
	return id.String()
}

func repeatedSessionServiceByte(value byte) []byte {
	key := make([]byte, 32)
	for index := range key {
		key[index] = value
	}
	return key
}
