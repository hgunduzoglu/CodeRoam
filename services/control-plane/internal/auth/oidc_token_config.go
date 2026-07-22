package auth

import (
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"
)

const maxOIDCAudienceBytes = 1024

type OIDCVerifierConfig struct {
	Issuer           string
	Audience         string
	JWKSURL          string
	SigningAlgorithm string
}

// Valid reports whether every configured OIDC trust anchor is bounded and explicit.
func (config OIDCVerifierConfig) Valid() bool {
	return config.valid()
}

func (config OIDCVerifierConfig) valid() bool {
	if _, err := NewOIDCIdentity(config.Issuer, "configured-subject"); err != nil {
		return false
	}
	if !validOIDCAudience(config.Audience) || !validOIDCJWKSURL(config.JWKSURL) {
		return false
	}
	switch config.SigningAlgorithm {
	case "RS256", "RS384", "RS512", "PS256", "PS384", "PS512",
		"ES256", "ES384", "ES512", "EdDSA":
		return true
	default:
		return false
	}
}

func validOIDCAudience(audience string) bool {
	return len(audience) > 0 && len(audience) <= maxOIDCAudienceBytes &&
		utf8.ValidString(audience) && strings.TrimSpace(audience) == audience &&
		!strings.ContainsFunc(audience, unicode.IsControl)
}

func validOIDCJWKSURL(value string) bool {
	if len(value) == 0 || len(value) > maxOIDCIssuerBytes || !utf8.ValidString(value) {
		return false
	}
	parsed, err := url.Parse(value)
	return err == nil && parsed.Scheme == "https" && parsed.Hostname() != "" &&
		parsed.User == nil && !parsed.ForceQuery && parsed.Fragment == "" &&
		validOIDCPort(parsed) && parsed.String() == value
}
