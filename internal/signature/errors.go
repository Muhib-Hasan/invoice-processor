package signature

import "fmt"

// Error codes for signature verification
const (
	ErrCodeNoSignature       = "NO_SIGNATURE"
	ErrCodeInvalidSignature  = "INVALID_SIGNATURE"
	ErrCodeCertExpired       = "CERT_EXPIRED"
	ErrCodeCertNotYetValid   = "CERT_NOT_YET_VALID"
	ErrCodeCertRevoked       = "CERT_REVOKED"
	ErrCodeChainInvalid      = "CHAIN_INVALID"
	ErrCodeUntrustedRoot     = "UNTRUSTED_ROOT"
	ErrCodeOCSPUnavailable   = "OCSP_UNAVAILABLE"
	ErrCodeTimestampInvalid  = "TIMESTAMP_INVALID"
	ErrCodeUnsupportedFormat = "UNSUPPORTED_FORMAT"
	ErrCodeToolUnavailable   = "TOOL_UNAVAILABLE"
)

// SignatureError represents signature verification errors
type SignatureError struct {
	Code    string
	Field   string
	Message string
	Cause   error
}

func (e *SignatureError) Error() string {
	if e.Field != "" && e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %s (%v)", e.Code, e.Field, e.Message, e.Cause)
	}
	if e.Field != "" {
		return fmt.Sprintf("[%s] %s: %s", e.Code, e.Field, e.Message)
	}
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s (%v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *SignatureError) Unwrap() error {
	return e.Cause
}

// NewSignatureError creates a new signature error
func NewSignatureError(code, field, message string, cause error) *SignatureError {
	return &SignatureError{
		Code:    code,
		Field:   field,
		Message: message,
		Cause:   cause,
	}
}

// Common error constructors

// ErrNoSignature returns error when no signature found in document
func ErrNoSignature() *SignatureError {
	return NewSignatureError(ErrCodeNoSignature, "", "no signature found in document", nil)
}

// ErrInvalidSignature returns error when signature validation fails
func ErrInvalidSignature(cause error) *SignatureError {
	return NewSignatureError(ErrCodeInvalidSignature, "signature", "signature validation failed", cause)
}

// ErrCertExpired returns error when certificate has expired
func ErrCertExpired(subject string) *SignatureError {
	return NewSignatureError(ErrCodeCertExpired, "certificate", fmt.Sprintf("certificate expired: %s", subject), nil)
}

// ErrCertNotYetValid returns error when certificate is not yet valid
func ErrCertNotYetValid(subject string) *SignatureError {
	return NewSignatureError(ErrCodeCertNotYetValid, "certificate", fmt.Sprintf("certificate not yet valid: %s", subject), nil)
}

// ErrCertRevoked returns error when certificate has been revoked
func ErrCertRevoked(subject string) *SignatureError {
	return NewSignatureError(ErrCodeCertRevoked, "certificate", fmt.Sprintf("certificate revoked: %s", subject), nil)
}

// ErrChainInvalid returns error when certificate chain is invalid
func ErrChainInvalid(cause error) *SignatureError {
	return NewSignatureError(ErrCodeChainInvalid, "chain", "certificate chain validation failed", cause)
}

// ErrUntrustedRoot returns error when root CA is not trusted
func ErrUntrustedRoot(issuer string) *SignatureError {
	return NewSignatureError(ErrCodeUntrustedRoot, "chain", fmt.Sprintf("root CA not trusted: %s", issuer), nil)
}

// ErrOCSPUnavailable returns error when OCSP check fails
func ErrOCSPUnavailable(cause error) *SignatureError {
	return NewSignatureError(ErrCodeOCSPUnavailable, "ocsp", "OCSP check unavailable", cause)
}

// ErrTimestampInvalid returns error when timestamp verification fails
func ErrTimestampInvalid(cause error) *SignatureError {
	return NewSignatureError(ErrCodeTimestampInvalid, "timestamp", "timestamp verification failed", cause)
}

// ErrUnsupportedFormat returns error for unsupported file formats
func ErrUnsupportedFormat(format string) *SignatureError {
	return NewSignatureError(ErrCodeUnsupportedFormat, "", fmt.Sprintf("unsupported format: %s", format), nil)
}

// ErrToolUnavailable returns error when external tool is not available
func ErrToolUnavailable(tool string) *SignatureError {
	return NewSignatureError(ErrCodeToolUnavailable, "", fmt.Sprintf("external tool not available: %s", tool), nil)
}
