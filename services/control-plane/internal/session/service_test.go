package session

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/auth"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/device"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/workspace"
	"github.com/jackc/pgx/v5"
)

const (
	serviceTestSessionID = "1123456789abcdef0123456789abcdef"
	serviceTestDeviceID  = "2123456789abcdef0123456789abcdef"
	serviceTestAgentID   = "3123456789abcdef0123456789abcdef"
	serviceTestProjectID = "4123456789abcdef0123456789abcdef"
)

type sessionServiceTxStub struct {
	pgx.Tx
	commitErr     error
	rollbackErr   error
	commitCalls   int
	rollbackCalls int
}

func (tx *sessionServiceTxStub) Commit(context.Context) error {
	tx.commitCalls++
	return tx.commitErr
}

func (tx *sessionServiceTxStub) Rollback(context.Context) error {
	tx.rollbackCalls++
	if tx.commitCalls > 0 && tx.commitErr == nil {
		return pgx.ErrTxClosed
	}
	return tx.rollbackErr
}

type sessionServiceStarterStub struct {
	tx    pgx.Tx
	err   error
	calls int
}

func (starter *sessionServiceStarterStub) Begin(context.Context) (pgx.Tx, error) {
	starter.calls++
	return starter.tx, starter.err
}

type sessionServiceDeviceStub struct {
	err   error
	calls int
	tx    pgx.Tx
	order *[]string
}

func (stub *sessionServiceDeviceStub) Authorize(
	_ context.Context,
	tx pgx.Tx,
	_ auth.Actor,
	_ string,
) error {
	stub.calls++
	stub.tx = tx
	if stub.order != nil {
		*stub.order = append(*stub.order, "device")
	}
	return stub.err
}

type sessionServiceWorkspaceStub struct {
	agentErr     error
	projectErr   error
	agentCalls   int
	projectCalls int
	agentTx      pgx.Tx
	projectTx    pgx.Tx
	order        *[]string
}

func (stub *sessionServiceWorkspaceStub) AuthorizeAgent(
	_ context.Context,
	tx pgx.Tx,
	_ auth.Actor,
	_ string,
) error {
	stub.agentCalls++
	stub.agentTx = tx
	if stub.order != nil {
		*stub.order = append(*stub.order, "agent")
	}
	return stub.agentErr
}

func (stub *sessionServiceWorkspaceStub) AuthorizeProject(
	_ context.Context,
	tx pgx.Tx,
	_ auth.Actor,
	_ string,
	_ string,
) error {
	stub.projectCalls++
	stub.projectTx = tx
	if stub.order != nil {
		*stub.order = append(*stub.order, "project")
	}
	return stub.projectErr
}

type sessionServiceCreatorStub struct {
	err      error
	calls    int
	tx       pgx.Tx
	metadata Session
	order    *[]string
}

func (stub *sessionServiceCreatorStub) CreateOrGet(
	_ context.Context,
	tx pgx.Tx,
	metadata Session,
) (Session, error) {
	stub.calls++
	stub.tx = tx
	stub.metadata = metadata
	if stub.order != nil {
		*stub.order = append(*stub.order, "create")
	}
	return metadata, stub.err
}

func TestNewServiceRequiresDependencies(t *testing.T) {
	tx := &sessionServiceTxStub{}
	starter := &sessionServiceStarterStub{tx: tx}
	devices := &sessionServiceDeviceStub{}
	workspaces := &sessionServiceWorkspaceStub{}
	sessions := &sessionServiceCreatorStub{}
	now := func() time.Time { return time.Date(2026, time.July, 19, 16, 0, 0, 0, time.UTC) }

	tests := map[string]struct {
		transactions transactionStarter
		devices      deviceAuthorizer
		workspaces   workspaceAuthorizer
		sessions     sessionCreator
		region       string
		now          func() time.Time
	}{
		"missing transactions": {devices: devices, workspaces: workspaces, sessions: sessions, region: "eu-central-1", now: now},
		"missing devices":      {transactions: starter, workspaces: workspaces, sessions: sessions, region: "eu-central-1", now: now},
		"missing workspaces":   {transactions: starter, devices: devices, sessions: sessions, region: "eu-central-1", now: now},
		"missing sessions":     {transactions: starter, devices: devices, workspaces: workspaces, region: "eu-central-1", now: now},
		"invalid region":       {transactions: starter, devices: devices, workspaces: workspaces, sessions: sessions, region: "EU-central-1", now: now},
		"missing clock":        {transactions: starter, devices: devices, workspaces: workspaces, sessions: sessions, region: "eu-central-1"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := NewService(
				test.transactions, test.devices, test.workspaces, test.sessions,
				test.region, test.now,
			); err == nil {
				t.Fatal("NewService() error = nil")
			}
		})
	}
}

func TestServiceStartUsesOneOrderedTransaction(t *testing.T) {
	order := make([]string, 0, 4)
	tx := &sessionServiceTxStub{}
	starter := &sessionServiceStarterStub{tx: tx}
	devices := &sessionServiceDeviceStub{order: &order}
	workspaces := &sessionServiceWorkspaceStub{order: &order}
	sessions := &sessionServiceCreatorStub{order: &order}
	startedAt := time.Date(2026, time.July, 19, 16, 0, 0, 123, time.FixedZone("test", 2*60*60))
	service := newSessionServiceForTest(t, starter, devices, workspaces, sessions, func() time.Time {
		return startedAt
	})
	owner := newSessionTestActor(t, "0123456789abcdef0123456789abcdef")

	started, err := service.Start(
		context.Background(), owner, serviceTestSessionID, serviceTestDeviceID,
		serviceTestAgentID, serviceTestProjectID,
	)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if starter.calls != 1 || tx.commitCalls != 1 || tx.rollbackCalls != 1 {
		t.Fatalf("transaction calls = begin %d, commit %d, rollback %d", starter.calls, tx.commitCalls, tx.rollbackCalls)
	}
	if devices.tx != tx || workspaces.agentTx != tx || workspaces.projectTx != tx || sessions.tx != tx {
		t.Fatal("Start() did not pass one transaction through every owning-module call")
	}
	if len(order) != 4 || order[0] != "device" || order[1] != "agent" || order[2] != "project" || order[3] != "create" {
		t.Fatalf("authorization order = %v", order)
	}
	if started.id.String() != serviceTestSessionID || started.deviceID.String() != serviceTestDeviceID ||
		started.agentID.String() != serviceTestAgentID || started.projectID.String() != serviceTestProjectID ||
		started.relayRegion != "eu-central-1" || !started.startedAt.Equal(startedAt) ||
		started.startedAt.Location() != time.UTC || sessions.metadata.id != started.id || !started.OwnedBy(owner) {
		t.Fatal("Start() did not return the exact committed metadata")
	}
}

func TestServiceStartReportsUnknownCommitOutcome(t *testing.T) {
	commitErr := errors.New("commit acknowledgement lost")
	tx := &sessionServiceTxStub{commitErr: commitErr}
	starter := &sessionServiceStarterStub{tx: tx}
	devices := &sessionServiceDeviceStub{}
	workspaces := &sessionServiceWorkspaceStub{}
	sessions := &sessionServiceCreatorStub{}
	service := newSessionServiceForTest(t, starter, devices, workspaces, sessions, time.Now)
	owner := newSessionTestActor(t, "0123456789abcdef0123456789abcdef")

	started, err := service.Start(
		context.Background(), owner, serviceTestSessionID, serviceTestDeviceID,
		serviceTestAgentID, serviceTestProjectID,
	)
	if !errors.Is(err, ErrSessionCommitOutcomeUnknown) || !errors.Is(err, commitErr) {
		t.Fatalf("Start() error = %v, want outcome unknown wrapping commit error", err)
	}
	if started != (Session{}) {
		t.Fatal("ambiguous Start() returned session metadata")
	}
	if tx.commitCalls != 1 || tx.rollbackCalls != 1 || sessions.calls != 1 {
		t.Fatalf(
			"ambiguous transaction calls = commit %d, rollback %d, create %d",
			tx.commitCalls, tx.rollbackCalls, sessions.calls,
		)
	}
}

func TestServiceStartFailsClosedAndRollsBack(t *testing.T) {
	databaseErr := errors.New("database unavailable")
	tests := map[string]struct {
		deviceErr  error
		agentErr   error
		projectErr error
		createErr  error
		want       error
		calls      [4]int
	}{
		"device denied":  {deviceErr: device.ErrDeviceAccessDenied, want: ErrSessionAccessDenied, calls: [4]int{1, 0, 0, 0}},
		"device invalid": {deviceErr: device.ErrInvalidDevice, want: ErrSessionAccessDenied, calls: [4]int{1, 0, 0, 0}},
		"agent denied":   {agentErr: workspace.ErrAgentAccessDenied, want: ErrSessionAccessDenied, calls: [4]int{1, 1, 0, 0}},
		"project denied": {projectErr: workspace.ErrProjectAccessDenied, want: ErrSessionAccessDenied, calls: [4]int{1, 1, 1, 0}},
		"device failure": {deviceErr: databaseErr, want: ErrSessionPersistenceUnavailable, calls: [4]int{1, 0, 0, 0}},
		"session ID conflict": {
			createErr: ErrSessionAccessDenied, want: ErrSessionAccessDenied, calls: [4]int{1, 1, 1, 1},
		},
		"create failure": {createErr: databaseErr, want: ErrSessionPersistenceUnavailable, calls: [4]int{1, 1, 1, 1}},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			tx := &sessionServiceTxStub{}
			starter := &sessionServiceStarterStub{tx: tx}
			devices := &sessionServiceDeviceStub{err: test.deviceErr}
			workspaces := &sessionServiceWorkspaceStub{agentErr: test.agentErr, projectErr: test.projectErr}
			sessions := &sessionServiceCreatorStub{err: test.createErr}
			service := newSessionServiceForTest(t, starter, devices, workspaces, sessions, time.Now)
			owner := newSessionTestActor(t, "0123456789abcdef0123456789abcdef")

			started, err := service.Start(
				context.Background(), owner, serviceTestSessionID, serviceTestDeviceID,
				serviceTestAgentID, serviceTestProjectID,
			)
			if !errors.Is(err, test.want) {
				t.Fatalf("Start() error = %v, want %v", err, test.want)
			}
			if started != (Session{}) {
				t.Fatal("failed Start() returned session metadata")
			}
			gotCalls := [4]int{devices.calls, workspaces.agentCalls, workspaces.projectCalls, sessions.calls}
			if gotCalls != test.calls {
				t.Fatalf("owning-module calls = %v, want %v", gotCalls, test.calls)
			}
			if tx.commitCalls != 0 || tx.rollbackCalls != 1 {
				t.Fatalf("transaction calls = commit %d, rollback %d", tx.commitCalls, tx.rollbackCalls)
			}
		})
	}
}

func TestServiceStartRejectsInvalidBoundariesBeforeTransaction(t *testing.T) {
	tx := &sessionServiceTxStub{}
	starter := &sessionServiceStarterStub{tx: tx}
	devices := &sessionServiceDeviceStub{}
	workspaces := &sessionServiceWorkspaceStub{}
	sessions := &sessionServiceCreatorStub{}
	service := newSessionServiceForTest(t, starter, devices, workspaces, sessions, time.Now)
	owner := newSessionTestActor(t, "0123456789abcdef0123456789abcdef")
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	tests := map[string]struct {
		ctx       context.Context
		actor     auth.Actor
		sessionID string
		deviceID  string
		agentID   string
		projectID string
		want      error
	}{
		"nil context": {actor: owner, sessionID: serviceTestSessionID, deviceID: serviceTestDeviceID,
			agentID: serviceTestAgentID, projectID: serviceTestProjectID, want: ErrSessionPersistenceUnavailable},
		"canceled context": {ctx: canceledCtx, actor: owner, sessionID: serviceTestSessionID,
			deviceID: serviceTestDeviceID, agentID: serviceTestAgentID,
			projectID: serviceTestProjectID, want: context.Canceled},
		"missing actor": {ctx: context.Background(), sessionID: serviceTestSessionID,
			deviceID: serviceTestDeviceID, agentID: serviceTestAgentID,
			projectID: serviceTestProjectID, want: ErrSessionAccessDenied},
		"invalid session ID": {ctx: context.Background(), actor: owner, sessionID: "invalid",
			deviceID: serviceTestDeviceID, agentID: serviceTestAgentID,
			projectID: serviceTestProjectID, want: ErrSessionAccessDenied},
		"invalid device ID": {ctx: context.Background(), actor: owner, sessionID: serviceTestSessionID,
			deviceID: "invalid", agentID: serviceTestAgentID,
			projectID: serviceTestProjectID, want: ErrSessionAccessDenied},
		"invalid agent ID": {ctx: context.Background(), actor: owner, sessionID: serviceTestSessionID,
			deviceID: serviceTestDeviceID, agentID: "invalid",
			projectID: serviceTestProjectID, want: ErrSessionAccessDenied},
		"invalid project ID": {ctx: context.Background(), actor: owner, sessionID: serviceTestSessionID,
			deviceID: serviceTestDeviceID, agentID: serviceTestAgentID,
			projectID: "invalid", want: ErrSessionAccessDenied},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			started, err := service.Start(
				test.ctx, test.actor, test.sessionID, test.deviceID, test.agentID, test.projectID,
			)
			if !errors.Is(err, test.want) {
				t.Fatalf("Start() error = %v, want %v", err, test.want)
			}
			if started != (Session{}) {
				t.Fatal("invalid Start() returned session metadata")
			}
		})
	}
	if starter.calls != 0 || devices.calls != 0 || workspaces.agentCalls != 0 || sessions.calls != 0 {
		t.Fatal("invalid boundaries reached persistence")
	}

	var nilService *Service
	if _, err := nilService.Start(
		context.Background(), owner, serviceTestSessionID, serviceTestDeviceID,
		serviceTestAgentID, serviceTestProjectID,
	); !errors.Is(err, ErrSessionPersistenceUnavailable) {
		t.Fatalf("nil Service Start() error = %v, want ErrSessionPersistenceUnavailable", err)
	}
}

func newSessionServiceForTest(
	t *testing.T,
	starter transactionStarter,
	devices deviceAuthorizer,
	workspaces workspaceAuthorizer,
	sessions sessionCreator,
	now func() time.Time,
) *Service {
	t.Helper()
	service, err := NewService(
		starter, devices, workspaces, sessions, "eu-central-1", now,
	)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	return service
}
