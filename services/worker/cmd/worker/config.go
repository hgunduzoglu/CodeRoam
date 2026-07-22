package main

import (
	"errors"
	"strings"
)

const workerProcessingEnabledEnvironment = "WORKER_PROCESSING_ENABLED"

type workerConfig struct {
	postgresDSN       string
	processingEnabled bool
}

func loadWorkerConfig(getenv func(string) string) (workerConfig, error) {
	if getenv == nil {
		return workerConfig{}, errors.New("worker environment reader is required")
	}
	dsn := strings.TrimSpace(getenv("POSTGRES_DSN"))
	if dsn == "" {
		return workerConfig{}, errors.New("worker PostgreSQL DSN is required")
	}
	enabled, err := parseWorkerProcessingEnabled(getenv(workerProcessingEnabledEnvironment))
	if err != nil {
		return workerConfig{}, err
	}
	return workerConfig{postgresDSN: dsn, processingEnabled: enabled}, nil
}

func parseWorkerProcessingEnabled(value string) (bool, error) {
	switch strings.TrimSpace(value) {
	case "", "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, errors.New("WORKER_PROCESSING_ENABLED must be true or false")
	}
}
