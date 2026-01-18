package model

import "fmt"

// ParseError represents parsing errors with provider context
type ParseError struct {
	Provider Provider
	Field    string
	Message  string
	Cause    error
}

func (e *ParseError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %s (%v)", e.Provider, e.Field, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s: %s", e.Provider, e.Field, e.Message)
}

func (e *ParseError) Unwrap() error {
	return e.Cause
}

// NewParseError creates a new parse error
func NewParseError(provider Provider, field, message string, cause error) *ParseError {
	return &ParseError{
		Provider: provider,
		Field:    field,
		Message:  message,
		Cause:    cause,
	}
}

// ValidationError represents validation failures
type ValidationError struct {
	Field   string
	Value   interface{}
	Rule    string
	Message string
}

func (e *ValidationError) Error() string {
	if e.Value != nil {
		return fmt.Sprintf("validation failed on %s: %s (value=%v, rule=%s)", e.Field, e.Message, e.Value, e.Rule)
	}
	return fmt.Sprintf("validation failed on %s: %s (rule=%s)", e.Field, e.Message, e.Rule)
}

// NewValidationError creates a new validation error
func NewValidationError(field string, value interface{}, rule, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Value:   value,
		Rule:    rule,
		Message: message,
	}
}

// ExtractionError represents extraction failures
type ExtractionError struct {
	Method  string
	Message string
	Cause   error
}

func (e *ExtractionError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("extraction failed [%s]: %s (%v)", e.Method, e.Message, e.Cause)
	}
	return fmt.Sprintf("extraction failed [%s]: %s", e.Method, e.Message)
}

func (e *ExtractionError) Unwrap() error {
	return e.Cause
}

// NewExtractionError creates a new extraction error
func NewExtractionError(method, message string, cause error) *ExtractionError {
	return &ExtractionError{
		Method:  method,
		Message: message,
		Cause:   cause,
	}
}
