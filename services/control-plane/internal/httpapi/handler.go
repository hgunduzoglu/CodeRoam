package httpapi

import "net/http"

// NewHandler composes the authenticated M2 control-plane routes. The supplied
// authenticator remains the single transport trust boundary for every route.
func NewHandler(
	authenticator actorAuthenticator,
	projects projectLister,
	sessions sessionStarter,
) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/v1/projects", NewProjectsHandler(authenticator, projects))
	mux.Handle("/v1/sessions", NewSessionsHandler(authenticator, sessions))
	return mux
}
