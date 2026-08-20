package main

import "testing"

func TestVersionOutput(t *testing.T) {
	originalVersion, originalCommit := version, commit
	t.Cleanup(func() {
		version, commit = originalVersion, originalCommit
	})
	version = "1.2.3"
	commit = "0123456789abcdef0123456789abcdef01234567"

	const want = "coderoam-agent 1.2.3 (0123456789abcdef0123456789abcdef01234567)"
	if got := versionOutput(); got != want {
		t.Fatalf("versionOutput() = %q, want %q", got, want)
	}
}
