package server

import (
	"time"

	"github.com/rezonia/invoice-processor/internal/model"
)

// ProcessResponse is the response for process endpoints
type ProcessResponse struct {
	Invoice    *model.Invoice `json:"invoice"`
	Method     string         `json:"method"`
	Confidence float64        `json:"confidence"`
	Warnings   []string       `json:"warnings,omitempty"`
}

// ValidationResponse is the response for validate endpoint
type ValidationResponse struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// InfoResponse is the response for info endpoint
type InfoResponse struct {
	Format   string `json:"format"`
	MimeType string `json:"mime_type"`
	Size     int    `json:"size"`
}

// ErrorResponse is the standard error response
type ErrorResponse struct {
	Error    string   `json:"error"`
	Details  string   `json:"details,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// VerifyResponse is the response for signature verification endpoint
type VerifyResponse struct {
	Valid          bool               `json:"valid"`
	SignatureFound bool               `json:"signature_found"`
	SignatureValid bool               `json:"signature_valid"`
	CertChainValid bool               `json:"cert_chain_valid"`
	NotRevoked     bool               `json:"not_revoked"`
	TimestampValid bool               `json:"timestamp_valid,omitempty"`
	Format         string             `json:"format,omitempty"`
	Signer         *SignerInfoOutput  `json:"signer,omitempty"`
	SignedAt       *time.Time         `json:"signed_at,omitempty"`
	Warnings       []string           `json:"warnings,omitempty"`
	Errors         []string           `json:"errors,omitempty"`
}

// SignerInfoOutput holds signer info for API response
type SignerInfoOutput struct {
	Name         string     `json:"name,omitempty"`
	Organization string     `json:"organization,omitempty"`
	SerialNumber string     `json:"serial_number,omitempty"`
	Issuer       string     `json:"issuer,omitempty"`
	ValidFrom    *time.Time `json:"valid_from,omitempty"`
	ValidTo      *time.Time `json:"valid_to,omitempty"`
}
