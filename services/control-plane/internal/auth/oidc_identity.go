package auth

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	maxOIDCIssuerBytes  = 2048
	maxOIDCSubjectBytes = 255
)

var ErrInvalidOIDCIdentity = errors.New("invalid OIDC identity")

type OIDCIdentity struct {
	issuer  string
	subject string
}

func NewOIDCIdentity(issuer, subject string) (OIDCIdentity, error) {
	parsedIssuer, err := url.Parse(issuer)
	if err != nil || len(issuer) == 0 || len(issuer) > maxOIDCIssuerBytes ||
		!utf8.ValidString(issuer) || parsedIssuer.Scheme != "https" || parsedIssuer.Host == "" ||
		parsedIssuer.User != nil || parsedIssuer.ForceQuery || parsedIssuer.RawQuery != "" ||
		parsedIssuer.Fragment != "" ||
		parsedIssuer.String() != issuer {
		return OIDCIdentity{}, fmt.Errorf("%w: issuer", ErrInvalidOIDCIdentity)
	}
	if len(subject) == 0 || len(subject) > maxOIDCSubjectBytes || !utf8.ValidString(subject) ||
		strings.TrimSpace(subject) != subject || strings.ContainsFunc(subject, unicode.IsControl) {
		return OIDCIdentity{}, fmt.Errorf("%w: subject", ErrInvalidOIDCIdentity)
	}
	return OIDCIdentity{issuer: issuer, subject: subject}, nil
}

func (identity OIDCIdentity) valid() bool {
	validated, err := NewOIDCIdentity(identity.issuer, identity.subject)
	return err == nil && validated == identity
}
