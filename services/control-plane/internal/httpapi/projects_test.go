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
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/workspace"
)

type projectListerStub struct {
	summaries []workspace.ProjectSummary
	err       error
	calls     int
	actor     auth.Actor
	limit     int
}

func (stub *projectListerStub) ListProjects(
	_ context.Context,
	actor auth.Actor,
	limit int,
) ([]workspace.ProjectSummary, error) {
	stub.calls++
	stub.actor = actor
	stub.limit = limit
	return stub.summaries, stub.err
}

func TestProjectsHandlerReturnsBoundedOwnerSummaries(t *testing.T) {
	actor := newHTTPAPIActor(t)
	authenticator := &transportAuthenticatorStub{actor: actor}
	projects := &projectListerStub{summaries: []workspace.ProjectSummary{{
		ID: "1123456789abcdef0123456789abcdef", EnvironmentID: "2123456789abcdef0123456789abcdef",
		AgentID: "3123456789abcdef0123456789abcdef", Name: "CodeRoam",
		EnvironmentName: "Development", CreatedAt: time.Date(2026, time.July, 19, 23, 30, 0, 0, time.UTC),
	}}}
	handler := NewProjectsHandler(authenticator, projects)
	request := httptest.NewRequest(http.MethodGet, "/v1/projects?limit=25", nil)
	request.Header.Set("Authorization", "Bearer opaque-evidence")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	body := response.Body.String()
	actorID, _ := actor.UserID()
	listedActorID, listedActorOK := projects.actor.UserID()
	if response.Code != http.StatusOK || projects.calls != 1 || projects.limit != 25 ||
		!listedActorOK || listedActorID != actorID ||
		!strings.Contains(body, `"id":"1123456789abcdef0123456789abcdef"`) ||
		!strings.Contains(body, `"createdAt":"2026-07-19T23:30:00Z"`) ||
		strings.Contains(body, "rootPath") {
		t.Fatalf("response = %d, calls %d, limit %d, body %q", response.Code, projects.calls, projects.limit, body)
	}
}

func TestProjectsHandlerRejectsInvalidQueriesBeforeListing(t *testing.T) {
	tests := []string{
		"limit=0", "limit=101", "limit=ten", "limit=1&limit=2", "other=1", "limit=1&other=2", "limit=%zz",
	}
	for _, rawQuery := range tests {
		t.Run(rawQuery, func(t *testing.T) {
			projects := &projectListerStub{}
			handler := NewProjectsHandler(&transportAuthenticatorStub{actor: newHTTPAPIActor(t)}, projects)
			request := httptest.NewRequest(http.MethodGet, "/v1/projects?"+rawQuery, nil)
			request.Header.Set("Authorization", "Bearer opaque-evidence")
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			if response.Code != http.StatusBadRequest || projects.calls != 0 ||
				!strings.Contains(response.Body.String(), `"code":"invalid_request"`) {
				t.Fatalf("response = %d, calls %d, body %q", response.Code, projects.calls, response.Body.String())
			}
		})
	}
}

func TestProjectsHandlerMapsFixedFailures(t *testing.T) {
	tests := map[string]struct {
		err        error
		wantStatus int
		wantCode   string
	}{
		"invalid":  {err: workspace.ErrInvalidProjectList, wantStatus: http.StatusBadRequest, wantCode: "invalid_request"},
		"denied":   {err: workspace.ErrProjectAccessDenied, wantStatus: http.StatusForbidden, wantCode: "access_denied"},
		"canceled": {err: context.Canceled, wantStatus: http.StatusRequestTimeout, wantCode: "request_terminated"},
		"unavailable": {
			err:        errors.New("database leaked /secret/root"),
			wantStatus: http.StatusServiceUnavailable, wantCode: "projects_unavailable",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			projects := &projectListerStub{err: test.err}
			handler := NewProjectsHandler(&transportAuthenticatorStub{actor: newHTTPAPIActor(t)}, projects)
			request := httptest.NewRequest(http.MethodGet, "/v1/projects", nil)
			request.Header.Set("Authorization", "Bearer opaque-evidence")
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			if response.Code != test.wantStatus || !strings.Contains(
				response.Body.String(), `"code":"`+test.wantCode+`"`,
			) || strings.Contains(response.Body.String(), "/secret/root") {
				t.Fatalf("response = %d, body %q", response.Code, response.Body.String())
			}
		})
	}
}

func TestProjectsHandlerRejectsInvalidListerOutput(t *testing.T) {
	valid := workspace.ProjectSummary{
		ID: "1123456789abcdef0123456789abcdef", EnvironmentID: "2123456789abcdef0123456789abcdef",
		AgentID: "3123456789abcdef0123456789abcdef", Name: "CodeRoam",
		EnvironmentName: "Development", CreatedAt: time.Date(2026, time.July, 19, 23, 30, 0, 0, time.UTC),
	}
	tests := map[string]struct {
		summaries []workspace.ProjectSummary
		limit     string
	}{
		"invalid summary":     {summaries: []workspace.ProjectSummary{{ID: "invalid"}}, limit: "50"},
		"more than requested": {summaries: []workspace.ProjectSummary{valid, valid}, limit: "1"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			handler := NewProjectsHandler(
				&transportAuthenticatorStub{actor: newHTTPAPIActor(t)},
				&projectListerStub{summaries: test.summaries},
			)
			request := httptest.NewRequest(http.MethodGet, "/v1/projects?limit="+test.limit, nil)
			request.Header.Set("Authorization", "Bearer opaque-evidence")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusServiceUnavailable || !strings.Contains(
				response.Body.String(), `"code":"projects_unavailable"`,
			) {
				t.Fatalf("response = %d, body %q", response.Code, response.Body.String())
			}
		})
	}
}

func TestProjectsHandlerRequiresAuthenticationAndDependencies(t *testing.T) {
	projects := &projectListerStub{}
	missingAuthentication := NewProjectsHandler(&transportAuthenticatorStub{actor: newHTTPAPIActor(t)}, projects)
	request := httptest.NewRequest(http.MethodGet, "/v1/projects", nil)
	response := httptest.NewRecorder()
	missingAuthentication.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized || projects.calls != 0 {
		t.Fatalf("unauthenticated response = %d, calls %d", response.Code, projects.calls)
	}

	var typedNilProjects *projectListerStub
	missingProjects := NewProjectsHandler(&transportAuthenticatorStub{actor: newHTTPAPIActor(t)}, typedNilProjects)
	request = httptest.NewRequest(http.MethodGet, "/v1/projects", nil)
	request.Header.Set("Authorization", "Bearer opaque-evidence")
	response = httptest.NewRecorder()
	missingProjects.ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("typed-nil projects response = %d", response.Code)
	}
}

func TestParseProjectListLimitDefaults(t *testing.T) {
	if limit, ok := parseProjectListLimit(""); !ok || limit != defaultProjectListLimit {
		t.Fatalf("parseProjectListLimit(empty) = (%d, %t)", limit, ok)
	}
}
