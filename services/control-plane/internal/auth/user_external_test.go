package auth_test

import (
	"errors"
	"testing"

	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/auth"
)

func TestUserIDParsing(t *testing.T) {
	const encoded = "0123456789abcdef0123456789abcdef"
	id, err := auth.ParseUserID(encoded)
	if err != nil {
		t.Fatalf("ParseUserID() error = %v", err)
	}
	if id.String() != encoded {
		t.Fatalf("String() = %q", id.String())
	}

	for _, value := range []string{
		"",
		"0123456789ABCDEF0123456789ABCDEF",
		"zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz",
	} {
		if _, err := auth.ParseUserID(value); !errors.Is(err, auth.ErrInvalidUser) {
			t.Fatalf("ParseUserID(%q) error = %v, want ErrInvalidUser", value, err)
		}
	}

	var zero auth.UserID
	if zero.String() != "" {
		t.Fatalf("zero UserID String() = %q", zero.String())
	}
	if _, err := auth.ParseUserID(zero.String()); !errors.Is(err, auth.ErrInvalidUser) {
		t.Fatalf("ParseUserID(zero) error = %v, want ErrInvalidUser", err)
	}
}
