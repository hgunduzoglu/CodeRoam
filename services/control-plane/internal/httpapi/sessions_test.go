package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/auth"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/session"
)

type sessionStarterStub struct {
	started session.Session
	err     error
	calls   int
	actor   auth.Actor
	ids     []string
}

func (stub *sessionStarterStub) Start(
	_ context.Context, actor auth.Actor, sessionID, deviceID, agentID, projectID string,
) (session.Session, error) {
	stub.calls++
	stub.actor = actor
	stub.ids = []string{sessionID, deviceID, agentID, projectID}
	return stub.started, stub.err
}

func TestSessionsHandlerStartsMetadataOnlySession(t *testing.T) {
	actor := newHTTPAPIActor(t)
	started, err := session.NewSession(
		actor, "1123456789abcdef0123456789abcdef", "2123456789abcdef0123456789abcdef",
		"3123456789abcdef0123456789abcdef", "4123456789abcdef0123456789abcdef",
		"eu-central-1", time.Date(2026, time.July, 20, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}
	sessions := &sessionStarterStub{started: started}
	handler := NewSessionsHandler(&transportAuthenticatorStub{actor: actor}, sessions)
	request := httptest.NewRequest(http.MethodPost, "/v1/sessions", strings.NewReader(
		`{"sessionId":"1123456789abcdef0123456789abcdef","deviceId":"2123456789abcdef0123456789abcdef",`+
			`"agentId":"3123456789abcdef0123456789abcdef","projectId":"4123456789abcdef0123456789abcdef"}`,
	))
	request.Header.Set("Authorization", "Bearer opaque-evidence")
	request.Header.Set("Content-Type", "application/json; charset=utf-8")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || sessions.calls != 1 || len(sessions.ids) != 4 ||
		!strings.Contains(response.Body.String(), `"capability":"metadata-only"`) ||
		strings.Contains(response.Body.String(), "ticket") {
		t.Fatalf("response = %d, calls %d, body %q", response.Code, sessions.calls, response.Body.String())
	}
}

func TestSessionsHandlerRejectsMalformedRequestsBeforeStart(t *testing.T) {
	valid := `{"sessionId":"1123456789abcdef0123456789abcdef","deviceId":"2123456789abcdef0123456789abcdef",` +
		`"agentId":"3123456789abcdef0123456789abcdef","projectId":"4123456789abcdef0123456789abcdef"}`
	tests := map[string]struct {
		contentType string
		body        string
		wantStatus  int
	}{
		"wrong media type": {contentType: "text/plain", body: valid, wantStatus: http.StatusUnsupportedMediaType},
		"unknown field":    {contentType: "application/json", body: strings.TrimSuffix(valid, "}") + `,"ticket":"forged"}`, wantStatus: http.StatusBadRequest},
		"duplicate id": {contentType: "application/json", body: strings.TrimSuffix(valid, "}") +
			`,"sessionId":"5123456789abcdef0123456789abcdef"}`, wantStatus: http.StatusBadRequest},
		"case alias": {contentType: "application/json", body: strings.TrimSuffix(valid, "}") +
			`,"SessionID":"5123456789abcdef0123456789abcdef"}`, wantStatus: http.StatusBadRequest},
		"trailing object": {contentType: "application/json", body: valid + `{}`, wantStatus: http.StatusBadRequest},
		"invalid id":      {contentType: "application/json", body: strings.Replace(valid, "1123456789abcdef0123456789abcdef", "invalid", 1), wantStatus: http.StatusBadRequest},
		"oversized":       {contentType: "application/json", body: valid + strings.Repeat(" ", maxSessionRequestBytes), wantStatus: http.StatusBadRequest},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			sessions := &sessionStarterStub{}
			handler := NewSessionsHandler(&transportAuthenticatorStub{actor: newHTTPAPIActor(t)}, sessions)
			request := httptest.NewRequest(http.MethodPost, "/v1/sessions", strings.NewReader(test.body))
			request.Header.Set("Authorization", "Bearer opaque-evidence")
			request.Header.Set("Content-Type", test.contentType)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.wantStatus || sessions.calls != 0 {
				t.Fatalf("response = %d, calls %d, body %q", response.Code, sessions.calls, response.Body.String())
			}
		})
	}
}

func TestSessionsHandlerMapsFixedFailures(t *testing.T) {
	validBody := `{"sessionId":"1123456789abcdef0123456789abcdef","deviceId":"2123456789abcdef0123456789abcdef",` +
		`"agentId":"3123456789abcdef0123456789abcdef","projectId":"4123456789abcdef0123456789abcdef"}`
	tests := map[string]struct {
		err        error
		wantStatus int
		wantCode   string
	}{
		"denied":          {err: session.ErrSessionAccessDenied, wantStatus: http.StatusForbidden, wantCode: "access_denied"},
		"unknown outcome": {err: session.ErrSessionCommitOutcomeUnknown, wantStatus: http.StatusServiceUnavailable, wantCode: "session_outcome_unknown"},
		"canceled":        {err: context.Canceled, wantStatus: http.StatusRequestTimeout, wantCode: "request_terminated"},
		"unavailable":     {err: errors.New("database leaked secret"), wantStatus: http.StatusServiceUnavailable, wantCode: "sessions_unavailable"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			handler := NewSessionsHandler(
				&transportAuthenticatorStub{actor: newHTTPAPIActor(t)}, &sessionStarterStub{err: test.err},
			)
			request := httptest.NewRequest(http.MethodPost, "/v1/sessions", strings.NewReader(validBody))
			request.Header.Set("Authorization", "Bearer opaque-evidence")
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.wantStatus || !strings.Contains(response.Body.String(), `"code":"`+test.wantCode+`"`) ||
				strings.Contains(response.Body.String(), "secret") {
				t.Fatalf("response = %d, body %q", response.Code, response.Body.String())
			}
		})
	}
}

func TestSessionsHandlerRejectsMismatchedServiceResult(t *testing.T) {
	actor := newHTTPAPIActor(t)
	started, err := session.NewSession(
		actor, "5123456789abcdef0123456789abcdef", "2123456789abcdef0123456789abcdef",
		"3123456789abcdef0123456789abcdef", "4123456789abcdef0123456789abcdef",
		"eu-central-1", time.Date(2026, time.July, 20, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}
	handler := NewSessionsHandler(&transportAuthenticatorStub{actor: actor}, &sessionStarterStub{started: started})
	request := httptest.NewRequest(http.MethodPost, "/v1/sessions", strings.NewReader(
		`{"sessionId":"1123456789abcdef0123456789abcdef","deviceId":"2123456789abcdef0123456789abcdef",`+
			`"agentId":"3123456789abcdef0123456789abcdef","projectId":"4123456789abcdef0123456789abcdef"}`,
	))
	request.Header.Set("Authorization", "Bearer opaque-evidence")
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable || !strings.Contains(
		response.Body.String(), `"code":"sessions_unavailable"`,
	) {
		t.Fatalf("response = %d, body %q", response.Code, response.Body.String())
	}
}
