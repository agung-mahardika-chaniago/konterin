package database

import (
	"errors"
	"fmt"
	"net/mail"
	"strings"
)

// NormalizeEmail lowercases and validates an email address.
func NormalizeEmail(raw string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return "", errors.New("email wajib diisi")
	}
	addr, err := mail.ParseAddress(value)
	if err != nil || addr.Address != value || !strings.Contains(value, "@") {
		return "", fmt.Errorf("email tidak valid: %s", raw)
	}
	if len(value) > 150 {
		return "", errors.New("email maksimal 150 karakter")
	}
	return value, nil
}

// ValidatePassword enforces the MVP password policy: at least 8 characters,
// containing a letter and a digit, and at most 72 bytes (bcrypt limit).
func ValidatePassword(raw string) error {
	if len(raw) < 8 {
		return errors.New("password minimal 8 karakter")
	}
	if len(raw) > 72 {
		return errors.New("password maksimal 72 karakter")
	}
	hasLetter, hasDigit := false, false
	for _, r := range raw {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z':
			hasLetter = true
		case r >= '0' && r <= '9':
			hasDigit = true
		}
	}
	if !hasLetter || !hasDigit {
		return errors.New("password harus mengandung huruf dan angka")
	}
	return nil
}
