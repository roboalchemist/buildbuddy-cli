package output

import (
	"encoding/json"
	"fmt"
	"io"
)

// Error type constants for structured error responses.
const (
	ErrTypeAuthFailed    = "auth_failed"
	ErrTypeNotFound      = "not_found"
	ErrTypeUsageError    = "usage_error"
	ErrTypeAPIError      = "api_error"
	ErrTypeInternalError = "internal_error"
)

// StructuredError represents a machine-readable error with a stable type field.
type StructuredError struct {
	Type     string `json:"error_type"`
	Message  string `json:"message"`
	ExitCode int    `json:"exit_code"`
}

func (e *StructuredError) Error() string {
	return e.Message
}

// WriteJSON serializes the error as JSON to the given writer.
func (e *StructuredError) WriteJSON(w io.Writer) {
	data, err := json.Marshal(e)
	if err != nil {
		fmt.Fprintf(w, `{"error_type":"internal_error","message":"failed to marshal error: %s","exit_code":1}`+"\n", err)
		return
	}
	fmt.Fprintln(w, string(data))
}

// NewError creates a StructuredError with the given type, message, and exit code.
func NewError(errType, message string, exitCode int) *StructuredError {
	return &StructuredError{Type: errType, Message: message, ExitCode: exitCode}
}

// NewUsageError creates a usage error (exit code 2) for invalid flags or arguments.
func NewUsageError(message string) *StructuredError {
	return NewError(ErrTypeUsageError, message, 2)
}

// NewAuthError creates an authentication error (exit code 1) for missing or invalid API keys.
func NewAuthError(message string) *StructuredError {
	return NewError(ErrTypeAuthFailed, message, 1)
}

// NewNotFoundError creates a not-found error (exit code 1) for missing resources.
func NewNotFoundError(message string) *StructuredError {
	return NewError(ErrTypeNotFound, message, 1)
}

// NewAPIError creates an API error (exit code 1) for failed API requests.
func NewAPIError(message string) *StructuredError {
	return NewError(ErrTypeAPIError, message, 1)
}

// NewInternalError creates an internal error (exit code 1) for unexpected failures.
func NewInternalError(message string) *StructuredError {
	return NewError(ErrTypeInternalError, message, 1)
}
