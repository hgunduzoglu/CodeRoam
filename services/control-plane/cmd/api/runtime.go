package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"time"

	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/auth"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/device"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/httpapi"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/session"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/workspace"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type runtimeDatabase interface {
	Begin(context.Context) (pgx.Tx, error)
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func newRuntimeHandler(
	database runtimeDatabase,
	config apiConfig,
	now func() time.Time,
	oidcTransport *http.Transport,
) (http.Handler, error) {
	if isNilRuntimeDatabase(database) {
		return nil, errors.New("control-plane runtime database is required")
	}
	if now == nil {
		return nil, errors.New("control-plane runtime clock is required")
	}
	authRepository, err := auth.NewRepository(database)
	if err != nil {
		return nil, err
	}
	tokenVerifier, err := auth.NewRemoteOIDCTokenVerifier(
		config.oidc, auth.NewOIDCJWKSCache(), oidcTransport,
	)
	if err != nil {
		return nil, err
	}
	identityVerifier, err := auth.NewOIDCIdentityVerifier(tokenVerifier, authRepository)
	if err != nil {
		return nil, err
	}
	authService, err := auth.NewService(authRepository, identityVerifier)
	if err != nil {
		return nil, err
	}
	deviceRepository, err := device.NewRepository(database, now)
	if err != nil {
		return nil, err
	}
	workspaceRepository, err := workspace.NewRepository(database, now)
	if err != nil {
		return nil, err
	}
	sessionService, err := session.NewService(
		database, deviceRepository, workspaceRepository, session.NewRepository(), config.relayRegion, now,
	)
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(response http.ResponseWriter, _ *http.Request) {
		checkedAt := now().UTC()
		if checkedAt.IsZero() {
			http.Error(response, "health unavailable", http.StatusServiceUnavailable)
			return
		}
		response.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(response).Encode(healthResponse{
			Service: "coderoam-control-plane",
			Status:  "ok",
			Time:    checkedAt.Format(time.RFC3339),
		})
	})
	mux.Handle("/v1/", httpapi.NewHandler(authService, workspaceRepository, sessionService))
	return mux, nil
}

func isNilRuntimeDatabase(database runtimeDatabase) bool {
	if database == nil {
		return true
	}
	value := reflect.ValueOf(database)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}
