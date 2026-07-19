package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"time"

	"github.com/hgunduzoglu/coderoam/packages/go/ids"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/auth"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/session"
)

const maxSessionRequestBytes = 4 * 1024

type sessionStarter interface {
	Start(context.Context, auth.Actor, string, string, string, string) (session.Session, error)
}

type startSessionRequest struct {
	SessionID string `json:"sessionId"`
	DeviceID  string `json:"deviceId"`
	AgentID   string `json:"agentId"`
	ProjectID string `json:"projectId"`
}

type sessionResponse struct {
	ID          string `json:"id"`
	DeviceID    string `json:"deviceId"`
	AgentID     string `json:"agentId"`
	ProjectID   string `json:"projectId"`
	RelayRegion string `json:"relayRegion"`
	StartedAt   string `json:"startedAt"`
	Capability  string `json:"capability"`
}

func NewSessionsHandler(authenticator actorAuthenticator, sessions sessionStarter) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/sessions", func(response http.ResponseWriter, request *http.Request) {
		serveStartSession(response, request, sessions)
	})
	return RequireActor(authenticator, mux)
}

func serveStartSession(response http.ResponseWriter, request *http.Request, sessions sessionStarter) {
	if request == nil || isNilDependency(sessions) {
		writeError(response, http.StatusServiceUnavailable, "sessions_unavailable")
		return
	}
	actor, ok := ActorFromContext(request.Context())
	if !ok {
		writeError(response, http.StatusForbidden, "access_denied")
		return
	}
	mediaType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeError(response, http.StatusUnsupportedMediaType, "unsupported_media_type")
		return
	}
	request.Body = http.MaxBytesReader(response, request.Body, maxSessionRequestBytes)
	decoder := json.NewDecoder(request.Body)
	payload, err := decodeStartSessionRequest(decoder)
	if err != nil {
		writeError(response, http.StatusBadRequest, "invalid_request")
		return
	}
	for _, encodedID := range []string{payload.SessionID, payload.DeviceID, payload.AgentID, payload.ProjectID} {
		if _, err := ids.Parse(encodedID); err != nil {
			writeError(response, http.StatusBadRequest, "invalid_request")
			return
		}
	}
	started, err := sessions.Start(
		request.Context(), actor, payload.SessionID, payload.DeviceID, payload.AgentID, payload.ProjectID,
	)
	if err != nil {
		switch {
		case errors.Is(err, session.ErrSessionAccessDenied):
			writeError(response, http.StatusForbidden, "access_denied")
		case errors.Is(err, session.ErrSessionCommitOutcomeUnknown):
			writeError(response, http.StatusServiceUnavailable, "session_outcome_unknown")
		case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
			writeError(response, http.StatusRequestTimeout, "request_terminated")
		default:
			writeError(response, http.StatusServiceUnavailable, "sessions_unavailable")
		}
		return
	}
	metadata, ok := started.MetadataFor(actor)
	if !ok || metadata.ID != payload.SessionID || metadata.DeviceID != payload.DeviceID ||
		metadata.AgentID != payload.AgentID || metadata.ProjectID != payload.ProjectID {
		writeError(response, http.StatusServiceUnavailable, "sessions_unavailable")
		return
	}
	response.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(response).Encode(sessionResponse{
		ID: metadata.ID, DeviceID: metadata.DeviceID, AgentID: metadata.AgentID,
		ProjectID: metadata.ProjectID, RelayRegion: metadata.RelayRegion,
		StartedAt: metadata.StartedAt.UTC().Format(time.RFC3339Nano), Capability: "metadata-only",
	})
}

func decodeStartSessionRequest(decoder *json.Decoder) (startSessionRequest, error) {
	if decoder == nil {
		return startSessionRequest{}, errors.New("session request decoder is required")
	}
	opening, err := decoder.Token()
	if err != nil || opening != json.Delim('{') {
		return startSessionRequest{}, errors.New("session request must be an object")
	}
	payload := startSessionRequest{}
	seen := make(map[string]bool, 4)
	for decoder.More() {
		token, err := decoder.Token()
		key, ok := token.(string)
		if err != nil || !ok || seen[key] {
			return startSessionRequest{}, errors.New("session request field is invalid")
		}
		seen[key] = true
		var value string
		if err := decoder.Decode(&value); err != nil {
			return startSessionRequest{}, errors.New("session request value is invalid")
		}
		switch key {
		case "sessionId":
			payload.SessionID = value
		case "deviceId":
			payload.DeviceID = value
		case "agentId":
			payload.AgentID = value
		case "projectId":
			payload.ProjectID = value
		default:
			return startSessionRequest{}, errors.New("session request field is unknown")
		}
	}
	closing, err := decoder.Token()
	if err != nil || closing != json.Delim('}') || len(seen) != 4 {
		return startSessionRequest{}, errors.New("session request is incomplete")
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return startSessionRequest{}, errors.New("session request has trailing data")
	}
	return payload, nil
}
