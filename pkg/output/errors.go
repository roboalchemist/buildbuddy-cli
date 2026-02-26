package output

import (
	"encoding/json"
	"fmt"
	"io"
)

const (
	ErrTypeAuthFailed       = "auth_failed"
	ErrTypeNotFound         = "not_found"
	ErrTypeUsageError       = "usage_error"
	ErrTypeAPIError         = "api_error"
	ErrTypeInternalError    = "internal_error"
	ErrTypeInvocationFailed = "invocation_failed"
	ErrTypeTimeout          = "timeout"
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

func (e *StructuredError) WriteJSON(w io.Writer) {
	data, err := json.Marshal(e)
	if err != nil {
		fmt.Fprintf(w, `{"error_type":"internal_error","message":"failed to marshal error: %s","exit_code":1}`+"\n", err)
		return
	}
	fmt.Fprintln(w, string(data))
}

func NewError(errType, message string, exitCode int) *StructuredError {
	return &StructuredError{Type: errType, Message: message, ExitCode: exitCode}
}

func NewUsageError(message string) *StructuredError {
	return NewError(ErrTypeUsageError, message, 2)
}

func NewAuthError(message string) *StructuredError {
	return NewError(ErrTypeAuthFailed, message, 1)
}

func NewNotFoundError(message string) *StructuredError {
	return NewError(ErrTypeNotFound, message, 1)
}

func NewAPIError(message string) *StructuredError {
	return NewError(ErrTypeAPIError, message, 1)
}

func NewInternalError(message string) *StructuredError {
	return NewError(ErrTypeInternalError, message, 1)
}

func NewInvocationFailedError(message string) *StructuredError {
	return NewError(ErrTypeInvocationFailed, message, 1)
}

func NewTimeoutError(message string) *StructuredError {
	return NewError(ErrTypeTimeout, message, 2)
}
