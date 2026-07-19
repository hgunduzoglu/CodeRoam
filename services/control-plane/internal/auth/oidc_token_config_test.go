package auth

import (
	"strings"
	"testing"
)

func TestOIDCVerifierConfigAcceptsExplicitTrustAnchors(t *testing.T) {
	config := OIDCVerifierConfig{
		Issuer:           "https://identity.example/realms/coderoam",
		Audience:         "coderoam-mobile",
		JWKSURL:          "https://keys.example/realms/coderoam/protocol/openid-connect/certs?version=1",
		SigningAlgorithm: "RS256",
	}
	if !config.valid() {
		t.Fatal("OIDCVerifierConfig.valid() = false, want true")
	}
}

func TestOIDCVerifierConfigRejectsInvalidTrustAnchors(t *testing.T) {
	valid := OIDCVerifierConfig{
		Issuer:           "https://identity.example/realms/coderoam",
		Audience:         "coderoam-mobile",
		JWKSURL:          "https://identity.example/realms/coderoam/protocol/openid-connect/certs",
		SigningAlgorithm: "RS256",
	}
	tests := map[string]func(*OIDCVerifierConfig){
		"empty issuer":            func(config *OIDCVerifierConfig) { config.Issuer = "" },
		"insecure issuer":         func(config *OIDCVerifierConfig) { config.Issuer = "http://identity.example" },
		"invalid issuer port":     func(config *OIDCVerifierConfig) { config.Issuer = "https://identity.example:99999" },
		"issuer query":            func(config *OIDCVerifierConfig) { config.Issuer += "?tenant=one" },
		"empty audience":          func(config *OIDCVerifierConfig) { config.Audience = "" },
		"padded audience":         func(config *OIDCVerifierConfig) { config.Audience = " mobile" },
		"control audience":        func(config *OIDCVerifierConfig) { config.Audience = "mobile\nother" },
		"oversized audience":      func(config *OIDCVerifierConfig) { config.Audience = strings.Repeat("a", maxOIDCAudienceBytes+1) },
		"empty JWKS URL":          func(config *OIDCVerifierConfig) { config.JWKSURL = "" },
		"insecure JWKS URL":       func(config *OIDCVerifierConfig) { config.JWKSURL = "http://identity.example/keys" },
		"JWKS URL without host":   func(config *OIDCVerifierConfig) { config.JWKSURL = "https://:443/keys" },
		"JWKS URL empty port":     func(config *OIDCVerifierConfig) { config.JWKSURL = "https://identity.example:/keys" },
		"JWKS URL invalid port":   func(config *OIDCVerifierConfig) { config.JWKSURL = "https://identity.example:99999/keys" },
		"JWKS URL zero port":      func(config *OIDCVerifierConfig) { config.JWKSURL = "https://identity.example:0/keys" },
		"JWKS URL credentials":    func(config *OIDCVerifierConfig) { config.JWKSURL = "https://user@identity.example/keys" },
		"JWKS URL empty query":    func(config *OIDCVerifierConfig) { config.JWKSURL = "https://identity.example/keys?" },
		"JWKS URL fragment":       func(config *OIDCVerifierConfig) { config.JWKSURL = "https://identity.example/keys#current" },
		"symmetric algorithm":     func(config *OIDCVerifierConfig) { config.SigningAlgorithm = "HS256" },
		"unsigned algorithm":      func(config *OIDCVerifierConfig) { config.SigningAlgorithm = "none" },
		"empty signing algorithm": func(config *OIDCVerifierConfig) { config.SigningAlgorithm = "" },
	}

	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			config := valid
			mutate(&config)
			if config.valid() {
				t.Fatal("OIDCVerifierConfig.valid() = true, want false")
			}
		})
	}
}

func TestOIDCVerifierConfigAcceptsOnlyApprovedAsymmetricAlgorithms(t *testing.T) {
	for _, algorithm := range []string{
		"RS256", "RS384", "RS512", "PS256", "PS384", "PS512",
		"ES256", "ES384", "ES512", "EdDSA",
	} {
		t.Run(algorithm, func(t *testing.T) {
			config := OIDCVerifierConfig{
				Issuer:           "https://identity.example",
				Audience:         "coderoam-mobile",
				JWKSURL:          "https://identity.example/keys",
				SigningAlgorithm: algorithm,
			}
			if !config.valid() {
				t.Fatal("OIDCVerifierConfig.valid() = false, want true")
			}
		})
	}
}
