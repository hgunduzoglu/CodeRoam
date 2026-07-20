package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/auth"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type runtimeDatabaseStub struct{}

func (*runtimeDatabaseStub) Begin(context.Context) (pgx.Tx, error) {
	panic("unexpected runtime database Begin")
}

func (*runtimeDatabaseStub) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	panic("unexpected runtime database Exec")
}

func (*runtimeDatabaseStub) QueryRow(context.Context, string, ...any) pgx.Row {
	panic("unexpected runtime database QueryRow")
}

func TestNewRuntimeHandlerActivatesHealthAndAuthenticatedRoutes(t *testing.T) {
	now := time.Date(2026, time.July, 20, 12, 0, 0, 0, time.UTC)
	handler, err := newRuntimeHandler(
		&runtimeDatabaseStub{}, validAPITestConfig(), func() time.Time { return now },
		http.DefaultTransport.(*http.Transport),
	)
	if err != nil {
		t.Fatalf("newRuntimeHandler() error = %v", err)
	}

	healthResponse := httptest.NewRecorder()
	handler.ServeHTTP(healthResponse, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if healthResponse.Code != http.StatusOK ||
		healthResponse.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("GET /healthz status = %d headers = %v", healthResponse.Code, healthResponse.Header())
	}
	projectResponse := httptest.NewRecorder()
	handler.ServeHTTP(projectResponse, httptest.NewRequest(http.MethodGet, "/v1/projects", nil))
	if projectResponse.Code != http.StatusUnauthorized ||
		projectResponse.Header().Get("WWW-Authenticate") != "Bearer" {
		t.Fatalf("GET /v1/projects status = %d headers = %v", projectResponse.Code, projectResponse.Header())
	}
}

func TestNewRuntimeHandlerRequiresEveryDependency(t *testing.T) {
	valid := validAPITestConfig()
	now := time.Now
	for name, test := range map[string]struct {
		database  runtimeDatabase
		config    apiConfig
		now       func() time.Time
		transport *http.Transport
	}{
		"nil database":       {config: valid, now: now, transport: http.DefaultTransport.(*http.Transport)},
		"typed nil database": {database: (*runtimeDatabaseStub)(nil), config: valid, now: now, transport: http.DefaultTransport.(*http.Transport)},
		"nil clock":          {database: &runtimeDatabaseStub{}, config: valid, transport: http.DefaultTransport.(*http.Transport)},
		"nil transport":      {database: &runtimeDatabaseStub{}, config: valid, now: now},
		"invalid OIDC": {
			database: &runtimeDatabaseStub{}, config: func() apiConfig {
				config := valid
				config.oidc.SigningAlgorithm = "none"
				return config
			}(), now: now, transport: http.DefaultTransport.(*http.Transport),
		},
		"invalid relay region": {
			database: &runtimeDatabaseStub{}, config: func() apiConfig {
				config := valid
				config.relayRegion = "INVALID"
				return config
			}(), now: now, transport: http.DefaultTransport.(*http.Transport),
		},
	} {
		t.Run(name, func(t *testing.T) {
			if handler, err := newRuntimeHandler(test.database, test.config, test.now, test.transport); err == nil || handler != nil {
				t.Fatalf("newRuntimeHandler() = (%v, %v), want (nil, non-nil error)", handler, err)
			}
		})
	}
}

func validAPITestConfig() apiConfig {
	return apiConfig{
		postgresDSN: "postgres://control-plane", httpAddress: ":8080", relayRegion: "local",
		oidc: auth.OIDCVerifierConfig{
			Issuer: "https://identity.example/realms/coderoam", Audience: "coderoam-mobile",
			JWKSURL: "https://identity.example/realms/coderoam/keys", SigningAlgorithm: "RS256",
		},
	}
}
