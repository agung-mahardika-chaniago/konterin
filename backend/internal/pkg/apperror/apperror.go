// Package apperror defines domain errors that map cleanly to HTTP responses.
//
// Every expected failure should be expressed as *Error so handlers can send a
// stable error code and an Indonesian message without leaking internals.
package apperror

import (
	"errors"
	"fmt"
	"net/http"
)

// Error is a domain error with a stable machine-readable code.
type Error struct {
	Status  int
	Code    string
	Message string
	Details any
	cause   error
}

func (e *Error) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %v", e.Code, e.cause)
	}
	return e.Code + ": " + e.Message
}

func (e *Error) Unwrap() error { return e.cause }

// New builds a domain error.
func New(status int, code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message}
}

// Wrap attaches an internal cause. The cause is never sent to clients.
func (e *Error) Wrap(err error) *Error {
	clone := *e
	clone.cause = err
	return &clone
}

// WithDetails returns a copy carrying structured details for the client.
func (e *Error) WithDetails(details any) *Error {
	clone := *e
	clone.Details = details
	return &clone
}

// As extracts an *Error from an error chain.
func As(err error) (*Error, bool) {
	var target *Error
	if errors.As(err, &target) {
		return target, true
	}
	return nil, false
}

// Common constructors.

func BadRequest(message string) *Error {
	return New(http.StatusBadRequest, "BAD_REQUEST", message)
}

func Unauthorized(code, message string) *Error {
	return New(http.StatusUnauthorized, code, message)
}

func InvalidCredentials() *Error {
	return Unauthorized("INVALID_CREDENTIALS", "Email atau password salah.")
}

func Forbidden(code, message string) *Error {
	return New(http.StatusForbidden, code, message)
}

func NotFound(message string) *Error {
	return New(http.StatusNotFound, "NOT_FOUND", message)
}

func Conflict(code, message string) *Error {
	return New(http.StatusConflict, code, message)
}

func Validation(fields map[string]string) *Error {
	return New(http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Data yang dikirim tidak valid.").
		WithDetails(map[string]any{"fields": fields})
}

func RateLimited(message string) *Error {
	return New(http.StatusTooManyRequests, "RATE_LIMITED", message)
}

func Internal(err error) *Error {
	e := New(http.StatusInternalServerError, "INTERNAL_ERROR", "Terjadi kesalahan pada server.")
	e.cause = err
	return e
}
