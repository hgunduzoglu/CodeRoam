package auth

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestNewUser(t *testing.T) {
	createdAt := time.Date(2026, time.July, 17, 14, 30, 0, 0, time.FixedZone("test", 3*60*60))
	user, err := NewUser(
		"0123456789abcdef0123456789abcdef",
		"  PERSON@Example.COM ",
		"  Ada Lovelace  ",
		createdAt,
	)
	if err != nil {
		t.Fatalf("NewUser() error = %v", err)
	}
	if user.id.String() != "0123456789abcdef0123456789abcdef" {
		t.Fatalf("id = %q", user.id.String())
	}
	if user.email != "PERSON@example.com" {
		t.Fatalf("email = %q", user.email)
	}
	if user.displayName != "Ada Lovelace" {
		t.Fatalf("displayName = %q", user.displayName)
	}
	if !user.createdAt.Equal(createdAt) || user.createdAt.Location() != time.UTC {
		t.Fatalf("createdAt = %v", user.createdAt)
	}

	secondUser, err := NewUser(
		"1123456789abcdef0123456789abcdef",
		"person@EXAMPLE.COM",
		"Grace Hopper",
		createdAt,
	)
	if err != nil {
		t.Fatalf("NewUser() second error = %v", err)
	}
	if user.email == secondUser.email {
		t.Fatalf("distinct local parts collapsed to %q", user.email)
	}

	emailAtLimit := strings.Repeat("a", 64) + "@" +
		strings.Repeat("b", 63) + "." +
		strings.Repeat("c", 63) + "." +
		strings.Repeat("d", 61)
	if len(emailAtLimit) != maxEmailBytes {
		t.Fatalf("emailAtLimit length = %d", len(emailAtLimit))
	}
	if _, err := NewUser(
		"2123456789abcdef0123456789abcdef",
		emailAtLimit,
		"Katherine Johnson",
		createdAt,
	); err != nil {
		t.Fatalf("NewUser() maximum-length email error = %v", err)
	}
	emailOverLimit := emailAtLimit + "e"

	tests := map[string]struct {
		id          string
		email       string
		displayName string
		createdAt   time.Time
	}{
		"empty id": {
			email:       "person@example.com",
			displayName: "Ada",
			createdAt:   createdAt,
		},
		"uppercase id": {
			id:          "0123456789ABCDEF0123456789ABCDEF",
			email:       "person@example.com",
			displayName: "Ada",
			createdAt:   createdAt,
		},
		"non hexadecimal id": {
			id:          "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz",
			email:       "person@example.com",
			displayName: "Ada",
			createdAt:   createdAt,
		},
		"malformed email": {
			id:          "0123456789abcdef0123456789abcdef",
			email:       "not-an-email",
			displayName: "Ada",
			createdAt:   createdAt,
		},
		"email with display syntax": {
			id:          "0123456789abcdef0123456789abcdef",
			email:       "Ada <person@example.com>",
			displayName: "Ada",
			createdAt:   createdAt,
		},
		"oversized email": {
			id:          "0123456789abcdef0123456789abcdef",
			email:       emailOverLimit,
			displayName: "Ada",
			createdAt:   createdAt,
		},
		"empty display name": {
			id:        "0123456789abcdef0123456789abcdef",
			email:     "person@example.com",
			createdAt: createdAt,
		},
		"oversized display name": {
			id:          "0123456789abcdef0123456789abcdef",
			email:       "person@example.com",
			displayName: strings.Repeat("a", maxDisplayNameRunes+1),
			createdAt:   createdAt,
		},
		"invalid display encoding": {
			id:          "0123456789abcdef0123456789abcdef",
			email:       "person@example.com",
			displayName: string([]byte{0xff}),
			createdAt:   createdAt,
		},
		"display name with control character": {
			id:          "0123456789abcdef0123456789abcdef",
			email:       "person@example.com",
			displayName: "Ada\nLovelace",
			createdAt:   createdAt,
		},
		"missing creation time": {
			id:          "0123456789abcdef0123456789abcdef",
			email:       "person@example.com",
			displayName: "Ada",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := NewUser(test.id, test.email, test.displayName, test.createdAt)
			if !errors.Is(err, ErrInvalidUser) {
				t.Fatalf("NewUser() error = %v, want ErrInvalidUser", err)
			}
		})
	}
}
