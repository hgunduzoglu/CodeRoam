package ids

import (
	"errors"
	"testing"
)

func TestNewReturnsParseableOpaqueID(t *testing.T) {
	id, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if len(id.String()) != encodedLength {
		t.Fatalf("New() length = %d, want %d", len(id.String()), encodedLength)
	}
	parsed, err := Parse(id.String())
	if err != nil {
		t.Fatalf("Parse(New()) error = %v", err)
	}
	if parsed != id {
		t.Fatal("Parse(New()) changed the opaque ID")
	}
}

func TestParseRejectsMalformedOpaqueID(t *testing.T) {
	for _, value := range []string{
		"",
		"0123456789ABCDEF0123456789ABCDEF",
		"zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz",
	} {
		if _, err := Parse(value); !errors.Is(err, ErrInvalid) {
			t.Fatalf("Parse(%q) error = %v, want ErrInvalid", value, err)
		}
	}
	if (ID{}).String() != "" {
		t.Fatal("zero ID returned a usable encoding")
	}
}
