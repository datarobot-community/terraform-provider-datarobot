package client

import "fmt"

// NotFoundError represents a custom error type for not found errors.
type NotFoundError struct {
	Resource   string
	InnerError error
}

// Error implements the error interface for NotFoundError.
func (e *NotFoundError) Error() string {
	if e.InnerError != nil {
		return fmt.Sprintf("%s not found: %v", e.Resource, e.InnerError)
	}
	return fmt.Sprintf("%s not found", e.Resource)
}

// Unwrap returns the inner error for NotFoundError.
func (e *NotFoundError) Unwrap() error {
	return e.InnerError
}

// NewNotFoundError creates a new NotFoundError.
func NewNotFoundError(resource string) *NotFoundError {
	return &NotFoundError{Resource: resource}
}

// WrapNotFoundError wraps an existing error into a NotFoundError.
func WrapNotFoundError(resource string, err error) *NotFoundError {
	return &NotFoundError{Resource: resource, InnerError: err}
}

// UnauthorizedError represents a custom error type for access denied errors.
type UnauthorizedError struct {
	Resource   string
	InnerError error
}

// Error implements the error interface for UnauthorizedError.
func (e *UnauthorizedError) Error() string {
	if e.InnerError != nil {
		return fmt.Sprintf("access denied to %s: %v", e.Resource, e.InnerError)
	}
	return fmt.Sprintf("access denied to %s", e.Resource)
}

// Unwrap returns the inner error for UnauthorizedError.
func (e *UnauthorizedError) Unwrap() error {
	return e.InnerError
}

// NewUnauthorizedError creates a new UnauthorizedError.
func NewUnauthorizedError(resource string) *UnauthorizedError {
	return &UnauthorizedError{Resource: resource}
}

// WrapUnauthorizedError wraps an existing error into an UnauthorizedError.
func WrapUnauthorizedError(resource string, err error) *UnauthorizedError {
	return &UnauthorizedError{Resource: resource, InnerError: err}
}

// GenericError represents a custom error type for generic errors.
type GenericError struct {
	Message    string
	InnerError error
}

// Error implements the error interface for GenericError.
func (e *GenericError) Error() string {
	if e.InnerError != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.InnerError)
	}
	return e.Message
}

// Unwrap returns the inner error for GenericError.
func (e *GenericError) Unwrap() error {
	return e.InnerError
}

// NewGenericError creates a new GenericError.
func NewGenericError(message string) *GenericError {
	return &GenericError{Message: message}
}

// WrapGenericError wraps an existing error into a GenericError.
func WrapGenericError(message string, err error) *GenericError {
	return &GenericError{Message: message, InnerError: err}
}
