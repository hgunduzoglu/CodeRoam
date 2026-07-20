package main

import (
	"strings"
	"testing"

	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/auth"
)

func TestLoadAPIConfig(t *testing.T) {
	environment := map[string]string{
		"POSTGRES_DSN":           " postgres://control-plane ",
		"RELAY_REGION":           "local",
		"OIDC_ISSUER":            "https://identity.example/realms/coderoam",
		"OIDC_AUDIENCE":          "coderoam-mobile",
		"OIDC_JWKS_URL":          "https://identity.example/realms/coderoam/keys",
		"OIDC_SIGNING_ALGORITHM": "RS256",
	}
	config, err := loadAPIConfig(func(key string) string { return environment[key] })
	if err != nil {
		t.Fatalf("loadAPIConfig() error = %v", err)
	}
	wantOIDC := auth.OIDCVerifierConfig{
		Issuer: environment["OIDC_ISSUER"], Audience: environment["OIDC_AUDIENCE"],
		JWKSURL: environment["OIDC_JWKS_URL"], SigningAlgorithm: environment["OIDC_SIGNING_ALGORITHM"],
	}
	if config.postgresDSN != "postgres://control-plane" || config.httpAddress != ":8080" ||
		config.relayRegion != "local" || config.oidc != wantOIDC {
		t.Fatalf("loadAPIConfig() = %+v, want bounded exact configuration", config)
	}
}

func TestLoadAPIConfigRequiresEveryTrustInput(t *testing.T) {
	valid := map[string]string{
		"POSTGRES_DSN":           "postgres://control-plane",
		"RELAY_REGION":           "local",
		"OIDC_ISSUER":            "https://identity.example/realms/coderoam",
		"OIDC_AUDIENCE":          "coderoam-mobile",
		"OIDC_JWKS_URL":          "https://identity.example/realms/coderoam/keys",
		"OIDC_SIGNING_ALGORITHM": "RS256",
	}
	for _, missing := range []string{
		"POSTGRES_DSN", "RELAY_REGION", "OIDC_ISSUER", "OIDC_AUDIENCE", "OIDC_JWKS_URL",
		"OIDC_SIGNING_ALGORITHM",
	} {
		t.Run(missing, func(t *testing.T) {
			environment := make(map[string]string, len(valid))
			for key, value := range valid {
				environment[key] = value
			}
			delete(environment, missing)
			if _, err := loadAPIConfig(func(key string) string { return environment[key] }); err == nil || !strings.Contains(err.Error(), missing) {
				t.Fatalf("loadAPIConfig() error = %v, want missing %s", err, missing)
			}
		})
	}
}

func TestLoadAPIConfigRejectsMissingEnvironmentReader(t *testing.T) {
	if _, err := loadAPIConfig(nil); err == nil {
		t.Fatal("loadAPIConfig(nil) error = nil")
	}
}

func TestLoadAPIConfigRejectsUnsafeHTTPAddress(t *testing.T) {
	valid := map[string]string{
		"POSTGRES_DSN":           "postgres://control-plane",
		"RELAY_REGION":           "local",
		"OIDC_ISSUER":            "https://identity.example/realms/coderoam",
		"OIDC_AUDIENCE":          "coderoam-mobile",
		"OIDC_JWKS_URL":          "https://identity.example/realms/coderoam/keys",
		"OIDC_SIGNING_ALGORITHM": "RS256",
	}
	for name, address := range map[string]string{
		"missing port":   "localhost",
		"zero port":      ":0",
		"invalid port":   ":99999",
		"control":        ":8080\nforged-log",
		"oversized host": strings.Repeat("a", maxAPIHTTPAddressBytes) + ":1",
	} {
		t.Run(name, func(t *testing.T) {
			environment := func(key string) string {
				if key == "CONTROL_PLANE_HTTP_ADDR" {
					return address
				}
				return valid[key]
			}
			if _, err := loadAPIConfig(environment); err == nil {
				t.Fatal("loadAPIConfig() error = nil")
			}
		})
	}
}
