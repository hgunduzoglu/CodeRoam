package main

import (
	"strings"
	"testing"
)

func TestLoadWorkerConfig(t *testing.T) {
	tests := map[string]struct {
		environment map[string]string
		want        workerConfig
		wantError   string
	}{
		"defaults processing on": {
			environment: map[string]string{"POSTGRES_DSN": " postgres://worker "},
			want:        workerConfig{postgresDSN: "postgres://worker", processingEnabled: true},
		},
		"processing disabled": {
			environment: map[string]string{
				"POSTGRES_DSN":                     "postgres://worker",
				workerProcessingEnabledEnvironment: "false",
			},
			want: workerConfig{postgresDSN: "postgres://worker", processingEnabled: false},
		},
		"missing DSN": {
			environment: map[string]string{},
			wantError:   "PostgreSQL DSN is required",
		},
		"invalid processing flag": {
			environment: map[string]string{
				"POSTGRES_DSN":                     "postgres://worker",
				workerProcessingEnabledEnvironment: "yes",
			},
			wantError: "must be true or false",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			getenv := func(key string) string { return test.environment[key] }
			got, err := loadWorkerConfig(getenv)
			if test.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantError) {
					t.Fatalf("loadWorkerConfig() error = %v, want %q", err, test.wantError)
				}
				return
			}
			if err != nil || got != test.want {
				t.Fatalf("loadWorkerConfig() = (%+v, %v), want (%+v, nil)", got, err, test.want)
			}
		})
	}
}

func TestLoadWorkerConfigRejectsMissingReader(t *testing.T) {
	if _, err := loadWorkerConfig(nil); err == nil {
		t.Fatal("loadWorkerConfig(nil) error = nil")
	}
}
