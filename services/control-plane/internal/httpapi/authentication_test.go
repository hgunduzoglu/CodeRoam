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
)

type transportAuthenticatorStub struct {
	actor    auth.Actor
	err      error
	calls    int
	evidence string
}

func (stub *transportAuthenticatorStub) Authenticate(
	_ context.Context,
	evidence string,
) (auth.Actor, error) {
	stub.calls++
	stub.evidence = evidence
	return stub.actor, stub.err
}

type transportVerifierStub struct {
	userID auth.UserID
}

func (stub transportVerifierStub) Verify(context.Context, string) (auth.UserID, error) {
	return stub.userID, nil
}

type transportUserFinderStub struct {
	user auth.User
}

func (stub transportUserFinderStub) FindByID(context.Context, auth.UserID) (auth.User, error) {
	return stub.user, nil
}

type nilTransportHandler struct{}

func (*nilTransportHandler) ServeHTTP(http.ResponseWriter, *http.Request) {
	panic("typed-nil handler was invoked")
}

func TestRequireActorAuthenticatesBearerEvidence(t *testing.T) {
	actor := newHTTPAPIActor(t)
	authenticator := &transportAuthenticatorStub{actor: actor}
	nextCalls := 0
	handler := RequireActor(authenticator, http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		nextCalls++
		got, ok := ActorFromContext(request.Context())
		gotID, gotIDOK := got.UserID()
		wantID, _ := actor.UserID()
		if !ok || !gotIDOK || gotID != wantID {
			t.Fatal("authenticated actor was not available to the handler")
		}
		response.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodGet, "/v1/projects", nil)
	request.Header.Set("Authorization", "bEaReR opaque-evidence")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent || nextCalls != 1 || authenticator.calls != 1 ||
		authenticator.evidence != "opaque-evidence" {
		t.Fatalf(
			"response = %d, next %d, auth calls %d, evidence %q",
			response.Code, nextCalls, authenticator.calls, authenticator.evidence,
		)
	}
}

func TestRequireActorRejectsMalformedAuthorizationBeforeAuthentication(t *testing.T) {
	tests := map[string][]string{
		"missing":                  nil,
		"duplicate":                {"Bearer first", "Bearer second"},
		"coalesced duplicate":      {"Bearer first, Bearer second"},
		"comma":                    {"Bearer first,second"},
		"wrong scheme":             {"Basic evidence"},
		"missing evidence":         {"Bearer "},
		"leading whitespace":       {" Bearer evidence"},
		"trailing whitespace":      {"Bearer evidence "},
		"internal whitespace":      {"Bearer first second"},
		"internal tab":             {"Bearer first\tsecond"},
		"control":                  {"Bearer first\x00second"},
		"characters after padding": {"Bearer first=second"},
		"padding only":             {"Bearer ="},
		"multiple padding only":    {"Bearer ==="},
	}
	for name, values := range tests {
		t.Run(name, func(t *testing.T) {
			authenticator := &transportAuthenticatorStub{}
			handler := RequireActor(authenticator, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				t.Fatal("unauthenticated request reached handler")
			}))
			request := httptest.NewRequest(http.MethodGet, "/v1/projects", nil)
			for _, value := range values {
				request.Header.Add("Authorization", value)
			}
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			if response.Code != http.StatusUnauthorized || authenticator.calls != 0 ||
				response.Header().Get("WWW-Authenticate") != "Bearer" ||
				!strings.Contains(response.Body.String(), `"code":"unauthenticated"`) {
				t.Fatalf("response = %d, headers %v, body %q", response.Code, response.Header(), response.Body.String())
			}
		})
	}
}

func TestRequireActorMapsAuthenticationFailuresToFixedErrors(t *testing.T) {
	tests := map[string]struct {
		actor      auth.Actor
		err        error
		wantStatus int
		wantCode   string
	}{
		"rejected": {
			err: auth.ErrUnauthenticated, wantStatus: http.StatusUnauthorized, wantCode: "unauthenticated",
		},
		"zero actor": {
			wantStatus: http.StatusUnauthorized, wantCode: "unauthenticated",
		},
		"unavailable": {
			err:        errors.New("provider leaked secret-token"),
			wantStatus: http.StatusServiceUnavailable, wantCode: "authentication_unavailable",
		},
		"canceled": {
			err: context.Canceled, wantStatus: http.StatusRequestTimeout, wantCode: "request_terminated",
		},
		"deadline": {
			err: context.DeadlineExceeded, wantStatus: http.StatusRequestTimeout, wantCode: "request_terminated",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			authenticator := &transportAuthenticatorStub{actor: test.actor, err: test.err}
			handler := RequireActor(authenticator, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				t.Fatal("failed authentication reached handler")
			}))
			request := httptest.NewRequest(http.MethodGet, "/v1/projects", nil)
			request.Header.Set("Authorization", "Bearer secret-token")
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			if response.Code != test.wantStatus || !strings.Contains(
				response.Body.String(), `"code":"`+test.wantCode+`"`,
			) || strings.Contains(response.Body.String(), "secret-token") {
				t.Fatalf("response = %d, body %q", response.Code, response.Body.String())
			}
		})
	}
}

func TestRequireActorRejectsMissingDependencies(t *testing.T) {
	var typedNilAuthenticator *transportAuthenticatorStub
	var typedNilHandler *nilTransportHandler
	tests := map[string]struct {
		authenticator actorAuthenticator
		next          http.Handler
	}{
		"nil authenticator": {
			next: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				t.Fatal("nil authenticator reached handler")
			}),
		},
		"typed nil authenticator": {
			authenticator: typedNilAuthenticator,
			next: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				t.Fatal("typed-nil authenticator reached handler")
			}),
		},
		"typed nil handler": {
			authenticator: &transportAuthenticatorStub{}, next: typedNilHandler,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/v1/projects", nil)
			response := httptest.NewRecorder()
			RequireActor(test.authenticator, test.next).ServeHTTP(response, request)
			if response.Code != http.StatusServiceUnavailable || !strings.Contains(
				response.Body.String(), `"code":"authentication_unavailable"`,
			) {
				t.Fatalf("dependency response = %d, body %q", response.Code, response.Body.String())
			}
		})
	}
}

func TestActorFromContextFailsClosed(t *testing.T) {
	if _, ok := ActorFromContext(nil); ok {
		t.Fatal("nil context returned an actor")
	}
	if _, ok := ActorFromContext(context.Background()); ok {
		t.Fatal("empty context returned an actor")
	}
	if _, ok := ActorFromContext(context.WithValue(context.Background(), actorContextKey{}, auth.Actor{})); ok {
		t.Fatal("zero actor was usable")
	}
}

func newHTTPAPIActor(t *testing.T) auth.Actor {
	t.Helper()
	userID, err := auth.ParseUserID("0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatalf("ParseUserID() error = %v", err)
	}
	user, err := auth.NewUser(
		userID.String(), "owner@example.com", "Owner",
		time.Date(2026, time.July, 19, 22, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("NewUser() error = %v", err)
	}
	service, err := auth.NewService(
		transportUserFinderStub{user: user}, transportVerifierStub{userID: userID},
	)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	actor, err := service.Authenticate(context.Background(), "opaque-evidence")
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	return actor
}
