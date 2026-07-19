package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"

	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/auth"
)

type actorAuthenticator interface {
	Authenticate(context.Context, string) (auth.Actor, error)
}

type actorContextKey struct{}

type errorEnvelope struct {
	Error struct {
		Code string `json:"code"`
	} `json:"error"`
}

func RequireActor(authenticator actorAuthenticator, next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if isNilDependency(authenticator) || isNilDependency(next) || request == nil {
			writeError(response, http.StatusServiceUnavailable, "authentication_unavailable")
			return
		}
		evidence, ok := bearerEvidence(request.Header.Values("Authorization"))
		if !ok {
			response.Header().Set("WWW-Authenticate", "Bearer")
			writeError(response, http.StatusUnauthorized, "unauthenticated")
			return
		}
		actor, err := authenticator.Authenticate(request.Context(), evidence)
		if err != nil {
			switch {
			case errors.Is(err, auth.ErrUnauthenticated):
				response.Header().Set("WWW-Authenticate", "Bearer")
				writeError(response, http.StatusUnauthorized, "unauthenticated")
			case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
				writeError(response, http.StatusRequestTimeout, "request_terminated")
			default:
				writeError(response, http.StatusServiceUnavailable, "authentication_unavailable")
			}
			return
		}
		if _, ok := actor.UserID(); !ok {
			response.Header().Set("WWW-Authenticate", "Bearer")
			writeError(response, http.StatusUnauthorized, "unauthenticated")
			return
		}
		next.ServeHTTP(response, request.WithContext(context.WithValue(
			request.Context(), actorContextKey{}, actor,
		)))
	})
}

func ActorFromContext(ctx context.Context) (auth.Actor, bool) {
	if ctx == nil {
		return auth.Actor{}, false
	}
	actor, ok := ctx.Value(actorContextKey{}).(auth.Actor)
	if !ok {
		return auth.Actor{}, false
	}
	if _, ok := actor.UserID(); !ok {
		return auth.Actor{}, false
	}
	return actor, true
}

func bearerEvidence(values []string) (string, bool) {
	if len(values) != 1 {
		return "", false
	}
	scheme, evidence, found := strings.Cut(values[0], " ")
	if !found || !strings.EqualFold(scheme, "Bearer") || evidence == "" {
		return "", false
	}
	padding := false
	tokenCharacters := false
	for _, character := range evidence {
		if character == '=' {
			if !tokenCharacters {
				return "", false
			}
			padding = true
			continue
		}
		allowed := character >= 'a' && character <= 'z' ||
			character >= 'A' && character <= 'Z' ||
			character >= '0' && character <= '9' ||
			strings.ContainsRune("-._~+/", character)
		if padding || !allowed {
			return "", false
		}
		tokenCharacters = true
	}
	return evidence, true
}

func isNilDependency(value any) bool {
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflected.IsNil()
	default:
		return false
	}
}

func writeError(response http.ResponseWriter, status int, code string) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)
	envelope := errorEnvelope{}
	envelope.Error.Code = code
	_ = json.NewEncoder(response).Encode(envelope)
}
