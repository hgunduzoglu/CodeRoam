package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	"github.com/hgunduzoglu/coderoam/packages/go/ids"
	"github.com/hgunduzoglu/coderoam/packages/go/postgresx"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/auth"
)

func TestRuntimeHandlerIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_TEST_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_TEST_DSN is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)
	pool, err := postgresx.OpenPool(ctx, dsn)
	if err != nil {
		t.Fatalf("OpenPool() error = %v", err)
	}
	t.Cleanup(pool.Close)

	now := time.Now().UTC().Truncate(time.Second)
	userID := newRuntimeIntegrationID(t)
	deviceID := newRuntimeIntegrationID(t)
	agentID := newRuntimeIntegrationID(t)
	environmentID := newRuntimeIntegrationID(t)
	projectID := newRuntimeIntegrationID(t)
	sessionID := newRuntimeIntegrationID(t)
	issuer, subject := "https://identity.example/realms/coderoam", "runtime-"+userID
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		for _, fixture := range []struct {
			statement string
			arguments []any
		}{
			{"DELETE FROM session.sessions WHERE id = $1", []any{sessionID}},
			{"DELETE FROM workspace.projects WHERE id = $1", []any{projectID}},
			{"DELETE FROM workspace.environments WHERE id = $1", []any{environmentID}},
			{"DELETE FROM workspace.agents WHERE id = $1", []any{agentID}},
			{"DELETE FROM device.devices WHERE id = $1", []any{deviceID}},
			{"DELETE FROM auth.oidc_identities WHERE issuer = $1 AND subject = $2 AND user_id = $3", []any{issuer, subject, userID}},
			{"DELETE FROM auth.users WHERE id = $1", []any{userID}},
		} {
			if _, err := pool.Exec(cleanupCtx, fixture.statement, fixture.arguments...); err != nil {
				t.Errorf("cleanup runtime integration fixture: %v", err)
			}
		}
	})
	publicKey := bytes.Repeat([]byte{0x42}, 32)
	fixtures := []struct {
		statement string
		arguments []any
	}{
		{"INSERT INTO auth.users (id, email, display_name, created_at) VALUES ($1, $2, $3, $4)",
			[]any{userID, userID + "@example.com", "Runtime User", now.Add(-5 * time.Minute)}},
		{"INSERT INTO auth.oidc_identities (issuer, subject, user_id, linked_at) VALUES ($1, $2, $3, $4)",
			[]any{issuer, subject, userID, now.Add(-4 * time.Minute)}},
		{"INSERT INTO device.devices (id, user_id, name, platform, static_public_key, public_key_fingerprint, paired_at) VALUES ($1, $2, $3, $4, $5, $6, $7)",
			[]any{deviceID, userID, "Runtime phone", "ios", publicKey, "device-" + deviceID, now.Add(-3 * time.Minute)}},
		{"INSERT INTO workspace.agents (id, user_id, name, static_public_key, public_key_fingerprint, version, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7)",
			[]any{agentID, userID, "Runtime agent", publicKey, "agent-" + agentID, "1.0.0", now.Add(-3 * time.Minute)}},
		{"INSERT INTO workspace.environments (id, user_id, agent_id, name, provider, created_at) VALUES ($1, $2, $3, $4, $5, $6)",
			[]any{environmentID, userID, agentID, "Runtime environment", "local", now.Add(-2 * time.Minute)}},
		{"INSERT INTO workspace.projects (id, user_id, environment_id, name, root_path, created_at) VALUES ($1, $2, $3, $4, $5, $6)",
			[]any{projectID, userID, environmentID, "Runtime project", "/workspace/runtime", now.Add(-time.Minute)}},
	}
	for _, fixture := range fixtures {
		if _, err := pool.Exec(ctx, fixture.statement, fixture.arguments...); err != nil {
			t.Fatalf("insert runtime integration fixture: %v", err)
		}
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey() error = %v", err)
	}
	jwks, err := json.Marshal(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{
		Key: &privateKey.PublicKey, KeyID: "runtime", Algorithm: "RS256", Use: "sig",
	}}})
	if err != nil {
		t.Fatalf("json.Marshal(JWKS) error = %v", err)
	}
	keyServer := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		_, _ = response.Write(jwks)
	}))
	t.Cleanup(keyServer.Close)
	trustedTransport, ok := keyServer.Client().Transport.(*http.Transport)
	if !ok {
		t.Fatal("test TLS transport is unavailable")
	}
	config := validAPITestConfig()
	config.oidc = auth.OIDCVerifierConfig{
		Issuer: issuer, Audience: "coderoam-mobile", JWKSURL: keyServer.URL, SigningAlgorithm: "RS256",
	}
	handler, err := newRuntimeHandler(pool, config, func() time.Time { return now }, trustedTransport)
	if err != nil {
		t.Fatalf("newRuntimeHandler() error = %v", err)
	}
	evidence := newRuntimeIntegrationToken(t, privateKey, issuer, subject, now)

	projectRequest := httptest.NewRequest(http.MethodGet, "/v1/projects", nil)
	projectRequest.Header.Set("Authorization", "Bearer "+evidence)
	projectResponse := httptest.NewRecorder()
	handler.ServeHTTP(projectResponse, projectRequest)
	projectBody := projectResponse.Body.String()
	if projectResponse.Code != http.StatusOK || !strings.Contains(projectBody, projectID) ||
		!strings.Contains(projectBody, "Runtime project") {
		t.Fatalf("GET /v1/projects status = %d body = %s", projectResponse.Code, projectBody)
	}

	sessionBody := fmt.Sprintf(
		`{"sessionId":%q,"deviceId":%q,"agentId":%q,"projectId":%q}`,
		sessionID, deviceID, agentID, projectID,
	)
	sessionRequest := httptest.NewRequest(http.MethodPost, "/v1/sessions", strings.NewReader(sessionBody))
	sessionRequest.Header.Set("Authorization", "Bearer "+evidence)
	sessionRequest.Header.Set("Content-Type", "application/json")
	sessionResponse := httptest.NewRecorder()
	handler.ServeHTTP(sessionResponse, sessionRequest)
	responseBody := sessionResponse.Body.String()
	if sessionResponse.Code != http.StatusOK || !strings.Contains(responseBody, sessionID) ||
		!strings.Contains(responseBody, `"capability":"metadata-only"`) {
		t.Fatalf("POST /v1/sessions status = %d body = %s", sessionResponse.Code, responseBody)
	}
	var persisted int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM session.sessions WHERE id = $1", sessionID).Scan(&persisted); err != nil {
		t.Fatalf("read persisted runtime session: %v", err)
	}
	if persisted != 1 {
		t.Fatalf("persisted runtime sessions = %d, want 1", persisted)
	}
}

func newRuntimeIntegrationID(t *testing.T) string {
	t.Helper()
	id, err := ids.New()
	if err != nil {
		t.Fatalf("ids.New() error = %v", err)
	}
	return id.String()
}

func newRuntimeIntegrationToken(
	t *testing.T,
	privateKey *rsa.PrivateKey,
	issuer string,
	subject string,
	now time.Time,
) string {
	t.Helper()
	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.RS256, Key: privateKey},
		(&jose.SignerOptions{}).WithType("JWT").WithHeader(jose.HeaderKey("kid"), "runtime"),
	)
	if err != nil {
		t.Fatalf("jose.NewSigner() error = %v", err)
	}
	payload, err := json.Marshal(map[string]any{
		"iss": issuer, "sub": subject, "aud": "coderoam-mobile",
		"iat": now.Add(-time.Minute).Unix(), "exp": now.Add(5 * time.Minute).Unix(),
	})
	if err != nil {
		t.Fatalf("json.Marshal(token) error = %v", err)
	}
	signed, err := signer.Sign(payload)
	if err != nil {
		t.Fatalf("Signer.Sign() error = %v", err)
	}
	compact, err := signed.CompactSerialize()
	if err != nil {
		t.Fatalf("CompactSerialize() error = %v", err)
	}
	return compact
}
