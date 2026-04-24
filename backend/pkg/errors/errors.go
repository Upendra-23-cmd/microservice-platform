// Package errors provides typed, domain-aware application errors.
// These wrap domain sentinel errors and carry HTTP/gRPC status semantics.
package errors

import (
	"errors"
	"fmt"
)

// Kind classifies an application error for transport-layer mapping.
type Kind uint8

const (
	KindInternal        Kind = iota
	KindNotFound
	KindAlreadyExists
	KindUnauthenticated
	KindPermissionDenied
	KindValidation
	KindConflict
	KindTimeout
)

// AppError is the standard application error.
type AppError struct {
	kind    Kind
	message string
	cause   error
}

func (e *AppError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %v", e.message, e.cause)
	}
	return e.message
}

func (e *AppError) Unwrap() error { return e.cause }
func (e *AppError) Kind() Kind    { return e.kind }

// ============================================================
// Constructors
// ============================================================

func Internal(op string, cause error) error {
	return &AppError{kind: KindInternal, message: fmt.Sprintf("internal error during %s", op), cause: cause}
}

func NotFound(entity string) error {
	return &AppError{kind: KindNotFound, message: fmt.Sprintf("%s not found", entity)}
}

func AlreadyExists(entity string) error {
	return &AppError{kind: KindAlreadyExists, message: fmt.Sprintf("%s already exists", entity)}
}

func Unauthenticated(msg string) error {
	return &AppError{kind: KindUnauthenticated, message: msg}
}

func PermissionDenied(msg string) error {
	return &AppError{kind: KindPermissionDenied, message: msg}
}

func Validation(msg string) error {
	return &AppError{kind: KindValidation, message: msg}
}

func Conflict(msg string) error {
	return &AppError{kind: KindConflict, message: msg}
}

func Timeout(op string) error {
	return &AppError{kind: KindTimeout, message: fmt.Sprintf("operation timed out: %s", op)}
}

// ============================================================
// Type Checkers
// ============================================================

func IsNotFound(err error) bool          { return kindOf(err) == KindNotFound }
func IsAlreadyExists(err error) bool     { return kindOf(err) == KindAlreadyExists }
func IsUnauthenticated(err error) bool   { return kindOf(err) == KindUnauthenticated }
func IsPermissionDenied(err error) bool  { return kindOf(err) == KindPermissionDenied }
func IsValidation(err error) bool        { return kindOf(err) == KindValidation }
func IsInternal(err error) bool          { return kindOf(err) == KindInternal }
func IsConflict(err error) bool          { return kindOf(err) == KindConflict }

func kindOf(err error) Kind {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.kind
	}
	return KindInternal
}
