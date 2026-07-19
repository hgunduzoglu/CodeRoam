package auth

import (
	"errors"
	"strings"
	"testing"
)

func TestNewOIDCIdentityPreservesExactStableIdentifier(t *testing.T) {
	identity, err := NewOIDCIdentity(
		"https://identity.example/realms/coderoam/",
		"Case-Sensitive_Subject-123",
	)
	if err != nil {
		t.Fatalf("NewOIDCIdentity() error = %v", err)
	}
	if identity.issuer != "https://identity.example/realms/coderoam/" ||
		identity.subject != "Case-Sensitive_Subject-123" || !identity.valid() {
		t.Fatal("NewOIDCIdentity() did not preserve the exact issuer and subject")
	}
}

func TestNewOIDCIdentityRejectsInvalidBoundaries(t *testing.T) {
	tests := map[string]struct {
		issuer  string
		subject string
	}{
		"empty issuer":        {issuer: "", subject: "subject"},
		"insecure issuer":     {issuer: "http://identity.example", subject: "subject"},
		"issuer without host": {issuer: "https://:443/realm", subject: "subject"},
		"issuer credentials":  {issuer: "https://user@identity.example", subject: "subject"},
		"empty issuer port":   {issuer: "https://identity.example:/realm", subject: "subject"},
		"invalid issuer port": {
			issuer: "https://identity.example:99999/realm", subject: "subject",
		},
		"zero issuer port":   {issuer: "https://identity.example:0/realm", subject: "subject"},
		"empty issuer query": {issuer: "https://identity.example?", subject: "subject"},
		"issuer query":       {issuer: "https://identity.example?tenant=one", subject: "subject"},
		"issuer fragment":    {issuer: "https://identity.example#tenant", subject: "subject"},
		"issuer whitespace":  {issuer: "https://identity.example/path with space", subject: "subject"},
		"oversized issuer":   {issuer: "https://" + strings.Repeat("a", maxOIDCIssuerBytes), subject: "subject"},
		"empty subject":      {issuer: "https://identity.example", subject: ""},
		"padded subject":     {issuer: "https://identity.example", subject: " subject"},
		"control subject":    {issuer: "https://identity.example", subject: "subject\nother"},
		"oversized subject":  {issuer: "https://identity.example", subject: strings.Repeat("a", maxOIDCSubjectBytes+1)},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			identity, err := NewOIDCIdentity(test.issuer, test.subject)
			if !errors.Is(err, ErrInvalidOIDCIdentity) || identity != (OIDCIdentity{}) {
				t.Fatalf("NewOIDCIdentity() = (%v, %v), want zero ErrInvalidOIDCIdentity", identity, err)
			}
		})
	}
}
