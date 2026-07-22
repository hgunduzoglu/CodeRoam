package auth

import (
	"encoding/hex"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	encodedUserIDLength = 32
	maxEmailBytes       = 254
	maxDisplayNameRunes = 128
)

var ErrInvalidUser = errors.New("invalid user")

type UserID struct {
	encoded string
}

type User struct {
	id          UserID
	email       string
	displayName string
	createdAt   time.Time
}

func NewUser(id, email, displayName string, createdAt time.Time) (User, error) {
	userID, err := ParseUserID(id)
	if err != nil {
		return User{}, err
	}

	normalizedEmail, ok := normalizeEmail(email)
	if !ok {
		return User{}, fmt.Errorf("%w: email", ErrInvalidUser)
	}

	displayName = strings.TrimSpace(displayName)
	if displayName == "" || !utf8.ValidString(displayName) ||
		strings.ContainsFunc(displayName, unicode.IsControl) ||
		utf8.RuneCountInString(displayName) > maxDisplayNameRunes {
		return User{}, fmt.Errorf("%w: display name", ErrInvalidUser)
	}
	if createdAt.IsZero() {
		return User{}, fmt.Errorf("%w: creation time", ErrInvalidUser)
	}

	return User{
		id:          userID,
		email:       normalizedEmail,
		displayName: displayName,
		createdAt:   createdAt.UTC(),
	}, nil
}

func ParseUserID(value string) (UserID, error) {
	if len(value) != encodedUserIDLength || value != strings.ToLower(value) {
		return UserID{}, fmt.Errorf("%w: id", ErrInvalidUser)
	}
	if _, err := hex.DecodeString(value); err != nil {
		return UserID{}, fmt.Errorf("%w: id", ErrInvalidUser)
	}
	return UserID{encoded: value}, nil
}

func (id UserID) String() string {
	return id.encoded
}

func normalizeEmail(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > maxEmailBytes || !utf8.ValidString(value) {
		return "", false
	}
	address, err := mail.ParseAddress(value)
	if err != nil || address.Address != value {
		return "", false
	}
	domainSeparator := strings.LastIndexByte(value, '@')
	if domainSeparator <= 0 || domainSeparator == len(value)-1 {
		return "", false
	}
	normalized := value[:domainSeparator+1] + strings.ToLower(value[domainSeparator+1:])
	if len(normalized) > maxEmailBytes {
		return "", false
	}
	return normalized, true
}
