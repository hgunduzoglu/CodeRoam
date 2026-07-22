package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlerRoutesAuthenticatedM2Requests(t *testing.T) {
	authenticator := &transportAuthenticatorStub{actor: newHTTPAPIActor(t)}
	projects := &projectListerStub{}
	sessions := &sessionStarterStub{}
	handler := NewHandler(authenticator, projects, sessions)

	projectRequest := httptest.NewRequest(http.MethodGet, "/v1/projects", nil)
	projectRequest.Header.Set("Authorization", "Bearer project-evidence")
	projectResponse := httptest.NewRecorder()
	handler.ServeHTTP(projectResponse, projectRequest)

	sessionRequest := httptest.NewRequest(http.MethodPost, "/v1/sessions", strings.NewReader("{}"))
	sessionRequest.Header.Set("Authorization", "Bearer session-evidence")
	sessionRequest.Header.Set("Content-Type", "application/json")
	sessionResponse := httptest.NewRecorder()
	handler.ServeHTTP(sessionResponse, sessionRequest)

	if projectResponse.Code != http.StatusOK || projects.calls != 1 || sessions.calls != 0 {
		t.Fatalf(
			"project response = %d, project calls = %d, session calls = %d",
			projectResponse.Code, projects.calls, sessions.calls,
		)
	}
	if sessionResponse.Code != http.StatusBadRequest || authenticator.calls != 2 {
		t.Fatalf(
			"session response = %d, authentication calls = %d",
			sessionResponse.Code, authenticator.calls,
		)
	}
}

func TestHandlerRejectsUnauthenticatedAndUnsupportedRequestsBeforeServices(t *testing.T) {
	authenticator := &transportAuthenticatorStub{actor: newHTTPAPIActor(t)}
	projects := &projectListerStub{}
	sessions := &sessionStarterStub{}
	handler := NewHandler(authenticator, projects, sessions)

	tests := []struct {
		name       string
		method     string
		target     string
		authorized bool
		wantStatus int
	}{
		{name: "projects require authentication", method: http.MethodGet, target: "/v1/projects", wantStatus: http.StatusUnauthorized},
		{name: "sessions require authentication", method: http.MethodPost, target: "/v1/sessions", wantStatus: http.StatusUnauthorized},
		{name: "projects authenticate before rejecting method", method: http.MethodPost, target: "/v1/projects", wantStatus: http.StatusUnauthorized},
		{name: "sessions authenticate before rejecting method", method: http.MethodGet, target: "/v1/sessions", wantStatus: http.StatusUnauthorized},
		{name: "projects reject wrong method", method: http.MethodPost, target: "/v1/projects", authorized: true, wantStatus: http.StatusMethodNotAllowed},
		{name: "sessions reject wrong method", method: http.MethodGet, target: "/v1/sessions", authorized: true, wantStatus: http.StatusMethodNotAllowed},
		{name: "unknown route", method: http.MethodGet, target: "/v1/unknown", authorized: true, wantStatus: http.StatusNotFound},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, test.target, nil)
			if test.authorized {
				request.Header.Set("Authorization", "Bearer opaque-evidence")
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("response = %d, want %d", response.Code, test.wantStatus)
			}
		})
	}
	if authenticator.calls != 2 || projects.calls != 0 || sessions.calls != 0 {
		t.Fatalf(
			"authentication calls = %d, project calls = %d, session calls = %d",
			authenticator.calls, projects.calls, sessions.calls,
		)
	}
}

func TestHandlerFailsClosedWithTypedNilAuthenticator(t *testing.T) {
	var authenticator *transportAuthenticatorStub
	projects := &projectListerStub{}
	sessions := &sessionStarterStub{}
	request := httptest.NewRequest(http.MethodGet, "/v1/projects", nil)
	request.Header.Set("Authorization", "Bearer opaque-evidence")
	response := httptest.NewRecorder()

	NewHandler(authenticator, projects, sessions).ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable || projects.calls != 0 || sessions.calls != 0 ||
		!strings.Contains(response.Body.String(), `"code":"authentication_unavailable"`) {
		t.Fatalf(
			"response = %d, project calls = %d, session calls = %d, body %q",
			response.Code, projects.calls, sessions.calls, response.Body.String(),
		)
	}
}
