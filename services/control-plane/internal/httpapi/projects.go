package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/auth"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/workspace"
)

const defaultProjectListLimit = 50

type projectLister interface {
	ListProjects(context.Context, auth.Actor, int) ([]workspace.ProjectSummary, error)
}

type projectResponse struct {
	ID              string `json:"id"`
	EnvironmentID   string `json:"environmentId"`
	AgentID         string `json:"agentId"`
	Name            string `json:"name"`
	EnvironmentName string `json:"environmentName"`
	CreatedAt       string `json:"createdAt"`
}

type projectsEnvelope struct {
	Projects []projectResponse `json:"projects"`
}

func NewProjectsHandler(authenticator actorAuthenticator, projects projectLister) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/projects", func(response http.ResponseWriter, request *http.Request) {
		serveProjectList(response, request, projects)
	})
	return RequireActor(authenticator, mux)
}

func serveProjectList(response http.ResponseWriter, request *http.Request, projects projectLister) {
	if request == nil || isNilDependency(projects) {
		writeError(response, http.StatusServiceUnavailable, "projects_unavailable")
		return
	}
	actor, ok := ActorFromContext(request.Context())
	if !ok {
		writeError(response, http.StatusForbidden, "access_denied")
		return
	}
	limit, ok := parseProjectListLimit(request.URL.RawQuery)
	if !ok {
		writeError(response, http.StatusBadRequest, "invalid_request")
		return
	}
	summaries, err := projects.ListProjects(request.Context(), actor, limit)
	if err != nil {
		switch {
		case errors.Is(err, workspace.ErrInvalidProjectList):
			writeError(response, http.StatusBadRequest, "invalid_request")
		case errors.Is(err, workspace.ErrProjectAccessDenied):
			writeError(response, http.StatusForbidden, "access_denied")
		case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
			writeError(response, http.StatusRequestTimeout, "request_terminated")
		default:
			writeError(response, http.StatusServiceUnavailable, "projects_unavailable")
		}
		return
	}
	projectResponses := make([]projectResponse, 0, len(summaries))
	if len(summaries) > limit {
		writeError(response, http.StatusServiceUnavailable, "projects_unavailable")
		return
	}
	for _, summary := range summaries {
		if !summary.Valid() {
			writeError(response, http.StatusServiceUnavailable, "projects_unavailable")
			return
		}
		projectResponses = append(projectResponses, projectResponse{
			ID: summary.ID, EnvironmentID: summary.EnvironmentID, AgentID: summary.AgentID,
			Name: summary.Name, EnvironmentName: summary.EnvironmentName,
			CreatedAt: summary.CreatedAt.UTC().Format(time.RFC3339Nano),
		})
	}
	response.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(response).Encode(projectsEnvelope{Projects: projectResponses})
}

func parseProjectListLimit(rawQuery string) (int, bool) {
	if rawQuery == "" {
		return defaultProjectListLimit, true
	}
	values, err := url.ParseQuery(rawQuery)
	if err != nil || len(values) != 1 {
		return 0, false
	}
	limits, ok := values["limit"]
	if !ok || len(limits) != 1 {
		return 0, false
	}
	limit, err := strconv.Atoi(limits[0])
	return limit, err == nil && limit >= 1 && limit <= 100
}
