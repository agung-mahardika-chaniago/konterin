// Package cryptoutil provides random token and password helpers.
package cryptoutil

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"math/big"
)

const (
	passwordAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789"
	letterAlphabet   = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
	digitAlphabet    = "23456789"
)

// RandomToken returns n cryptographically random bytes encoded as URL-safe
// base64 without padding. 32 bytes yields the 256-bit refresh token required
// by the security specification.
func RandomToken(n int) (string, error) {
	if n < 16 {
		return "", errors.New("panjang token minimal 16 byte")
	}
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// HashToken returns the lowercase hex SHA-256 digest used for token storage.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// RandomPassword generates a temporary password that satisfies the password
// policy (letters and digits, no ambiguous characters).
func RandomPassword(length int) (string, error) {
	if length < 8 {
		length = 8
	}
	out := make([]byte, 0, length)
	for i := 0; i < length; i++ {
		ch, err := randomChar(passwordAlphabet)
		if err != nil {
			return "", err
		}
		out = append(out, ch)
	}
	// Guarantee at least one letter and one digit.
	if ch, err := randomChar(letterAlphabet); err != nil {
		return "", err
	} else {
		out[0] = ch
	}
	if ch, err := randomChar(digitAlphabet); err != nil {
		return "", err
	} else {
		out[1] = ch
	}
	return string(out), nil
}

func randomChar(alphabet string) (byte, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
	if err != nil {
		return 0, err
	}
	return alphabet[n.Int64()], nil
}
