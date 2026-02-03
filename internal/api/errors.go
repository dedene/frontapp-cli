package api

import (
	"errors"
	"fmt"
	"net/http"
)

const (
	ExitSuccess   = 0
	ExitError     = 1
	ExitUsage     = 2
	ExitAuth      = 3
	ExitNotFound  = 4
	ExitRateLimit = 5
)

var (
	ErrNotAuthenticated = errors.New("not authenticated")
	ErrRateLimited      = errors.New("rate limit exceeded")
	ErrNotFound         = errors.New("not found")
)

type APIError struct {
	StatusCode       int
	Message          string
	Details          string
	RequestedID      string // The ID used in the request (for hint generation)
	ExpectedResource string // Expected resource type (e.g., "conversation")
}

func (e *APIError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s", e.Message, e.Details)
	}

	return e.Message
}

func (e *APIError) ExitCode() int {
	switch e.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return ExitAuth
	case http.StatusNotFound:
		return ExitNotFound
	case http.StatusTooManyRequests:
		return ExitRateLimit
	default:
		return ExitError
	}
}

type CircuitBreakerError struct{}

func (e *CircuitBreakerError) Error() string {
	return "circuit breaker is open: too many consecutive failures"
}

type AuthError struct {
	Err error
}

func (e *AuthError) Error() string {
	return fmt.Sprintf("authentication error: %v", e.Err)
}

func (e *AuthError) Unwrap() error {
	return e.Err
}

type NotFoundError struct {
	Resource string
	ID       string
}

func (e *NotFoundError) Error() string {
	if e.ID != "" {
		return fmt.Sprintf("%s '%s' not found", e.Resource, e.ID)
	}

	return fmt.Sprintf("%s not found", e.Resource)
}

type RateLimitError struct {
	RetryAfter int // seconds
}

func (e *RateLimitError) Error() string {
	if e.RetryAfter > 0 {
		return fmt.Sprintf("rate limit exceeded, retry after %d seconds", e.RetryAfter)
	}

	return "rate limit exceeded"
}

// WrongResourceTypeError indicates the user provided an ID of the wrong resource type.
type WrongResourceTypeError struct {
	ExpectedType string // e.g., "conversation"
	ActualType   string // e.g., "message"
	ID           string // The ID that was provided
}

func (e *WrongResourceTypeError) Error() string {
	return fmt.Sprintf("'%s' is a %s ID, but a %s ID was expected", e.ID, e.ActualType, e.ExpectedType)
}
