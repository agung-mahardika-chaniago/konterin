package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
	"strings"
)

const credentialKeyVersion = 1

func encryptCredential(key []byte, plaintext string) ([]byte, error) {
	if len(key) != 32 {
		return nil, errors.New("kunci enkripsi tidak valid")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, []byte(plaintext), nil), nil
}

func decryptCredential(key, payload []byte) (string, error) {
	if len(key) != 32 {
		return "", errors.New("kunci enkripsi tidak valid")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(payload) < gcm.NonceSize() {
		return "", errors.New("kredensial tidak valid")
	}
	nonce, ciphertext := payload[:gcm.NonceSize()], payload[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func validCredentialType(value string) bool {
	switch value {
	case CredentialPIN, CredentialPassword, CredentialPattern:
		return true
	default:
		return false
	}
}

func validCredentialValue(value string) bool {
	trimmed := strings.TrimSpace(value)
	return trimmed != "" && len(trimmed) <= 200
}
