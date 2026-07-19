package auth

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
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
	if len(issuer) == 0 || len(issuer) > maxOIDCIssuerBytes || !utf8.ValidString(issuer) {
		return OIDCIdentity{}, fmt.Errorf("%w: issuer", ErrInvalidOIDCIdentity)
	}
	parsedIssuer, err := url.Parse(issuer)
	if err != nil || parsedIssuer.Scheme != "https" || parsedIssuer.Hostname() == "" ||
		parsedIssuer.User != nil || parsedIssuer.ForceQuery || parsedIssuer.RawQuery != "" ||
		parsedIssuer.Fragment != "" || !validOIDCPort(parsedIssuer) ||
		parsedIssuer.String() != issuer {
		return OIDCIdentity{}, fmt.Errorf("%w: issuer", ErrInvalidOIDCIdentity)
	}
	if len(subject) == 0 || len(subject) > maxOIDCSubjectBytes || !utf8.ValidString(subject) ||
		strings.TrimSpace(subject) != subject || strings.ContainsFunc(subject, unicode.IsControl) {
		return OIDCIdentity{}, fmt.Errorf("%w: subject", ErrInvalidOIDCIdentity)
	}
	return OIDCIdentity{issuer: issuer, subject: subject}, nil
}

func validOIDCPort(parsed *url.URL) bool {
	if parsed == nil || strings.HasSuffix(parsed.Host, ":") {
		return false
	}
	port := parsed.Port()
	if port == "" {
		return true
	}
	number, err := strconv.ParseUint(port, 10, 16)
	return err == nil && number != 0
}

func (identity OIDCIdentity) valid() bool {
	validated, err := NewOIDCIdentity(identity.issuer, identity.subject)
	return err == nil && validated == identity
}
