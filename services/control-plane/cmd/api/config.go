package main

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/auth"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/session"
)

type apiConfig struct {
	postgresDSN string
	httpAddress string
	relayRegion string
	oidc        auth.OIDCVerifierConfig
}

const maxAPIHTTPAddressBytes = 1024

func loadAPIConfig(getenv func(string) string) (apiConfig, error) {
	if getenv == nil {
		return apiConfig{}, errors.New("control-plane environment reader is required")
	}
	postgresDSN := strings.TrimSpace(getenv("POSTGRES_DSN"))
	if postgresDSN == "" {
		return apiConfig{}, errors.New("control-plane POSTGRES_DSN is required")
	}
	httpAddress := strings.TrimSpace(getenv("CONTROL_PLANE_HTTP_ADDR"))
	if httpAddress == "" {
		httpAddress = ":8080"
	}
	if !validAPIHTTPAddress(httpAddress) {
		return apiConfig{}, errors.New("control-plane CONTROL_PLANE_HTTP_ADDR is invalid")
	}
	relayRegion, err := requiredAPIEnvironment(getenv, "RELAY_REGION")
	if err != nil {
		return apiConfig{}, err
	}
	issuer, err := requiredAPIEnvironment(getenv, "OIDC_ISSUER")
	if err != nil {
		return apiConfig{}, err
	}
	audience, err := requiredAPIEnvironment(getenv, "OIDC_AUDIENCE")
	if err != nil {
		return apiConfig{}, err
	}
	jwksURL, err := requiredAPIEnvironment(getenv, "OIDC_JWKS_URL")
	if err != nil {
		return apiConfig{}, err
	}
	algorithm, err := requiredAPIEnvironment(getenv, "OIDC_SIGNING_ALGORITHM")
	if err != nil {
		return apiConfig{}, err
	}
	oidc := auth.OIDCVerifierConfig{
		Issuer: issuer, Audience: audience, JWKSURL: jwksURL, SigningAlgorithm: algorithm,
	}
	if !oidc.Valid() {
		return apiConfig{}, errors.New("control-plane OIDC configuration is invalid")
	}
	if !session.ValidRelayRegion(relayRegion) {
		return apiConfig{}, errors.New("control-plane RELAY_REGION is invalid")
	}
	return apiConfig{
		postgresDSN: postgresDSN,
		httpAddress: httpAddress,
		relayRegion: relayRegion,
		oidc:        oidc,
	}, nil
}

func validAPIHTTPAddress(value string) bool {
	if len(value) == 0 || len(value) > maxAPIHTTPAddressBytes || !utf8.ValidString(value) ||
		strings.TrimSpace(value) != value || strings.ContainsFunc(value, unicode.IsControl) {
		return false
	}
	_, encodedPort, err := net.SplitHostPort(value)
	if err != nil {
		return false
	}
	port, err := strconv.Atoi(encodedPort)
	return err == nil && port >= 1 && port <= 65535
}

func requiredAPIEnvironment(getenv func(string) string, key string) (string, error) {
	value := getenv(key)
	if value == "" {
		return "", fmt.Errorf("control-plane %s is required", key)
	}
	return value, nil
}
