package signature

import (
	"crypto/x509"
	"time"
)

// VerificationResult contains the complete signature verification outcome
type VerificationResult struct {
	// Overall validity - true only if all checks pass
	Valid bool `json:"valid"`

	// Individual check results
	SignatureFound bool `json:"signature_found"`
	SignatureValid bool `json:"signature_valid"`
	CertChainValid bool `json:"cert_chain_valid"`
	NotRevoked     bool `json:"not_revoked"`
	TimestampValid bool `json:"timestamp_valid,omitempty"`

	// Signer information
	Signer *SignerInfo `json:"signer,omitempty"`

	// Signing timestamp
	SignedAt *time.Time `json:"signed_at,omitempty"`

	// Certificate chain (not serialized to JSON)
	CertChain []*x509.Certificate `json:"-"`

	// Warnings (non-fatal issues)
	Warnings []string `json:"warnings,omitempty"`

	// Errors (reasons for invalid result)
	Errors []string `json:"errors,omitempty"`

	// Format of the verified document
	Format string `json:"format,omitempty"`
}

// SignerInfo contains certificate subject information
type SignerInfo struct {
	// Common name (CN)
	Name string `json:"name"`

	// Organization (O)
	Organization string `json:"organization,omitempty"`

	// Certificate serial number
	SerialNumber string `json:"serial_number"`

	// Issuer common name
	Issuer string `json:"issuer"`

	// Certificate validity period
	ValidFrom time.Time `json:"valid_from"`
	ValidTo   time.Time `json:"valid_to"`
}

// NewVerificationResult creates a new empty result
func NewVerificationResult() *VerificationResult {
	return &VerificationResult{
		Warnings: make([]string, 0),
		Errors:   make([]string, 0),
	}
}

// AddWarning adds a warning message to the result
func (r *VerificationResult) AddWarning(msg string) {
	r.Warnings = append(r.Warnings, msg)
}

// AddError adds an error message and sets Valid to false
func (r *VerificationResult) AddError(msg string) {
	r.Errors = append(r.Errors, msg)
	r.Valid = false
}

// SetSigner populates SignerInfo from an x509 certificate
func (r *VerificationResult) SetSigner(cert *x509.Certificate) {
	if cert == nil {
		return
	}

	signer := &SignerInfo{
		SerialNumber: cert.SerialNumber.String(),
		ValidFrom:    cert.NotBefore,
		ValidTo:      cert.NotAfter,
	}

	// Extract CN from Subject
	if len(cert.Subject.CommonName) > 0 {
		signer.Name = cert.Subject.CommonName
	}

	// Extract Organization
	if len(cert.Subject.Organization) > 0 {
		signer.Organization = cert.Subject.Organization[0]
	}

	// Extract Issuer CN
	if len(cert.Issuer.CommonName) > 0 {
		signer.Issuer = cert.Issuer.CommonName
	} else if len(cert.Issuer.Organization) > 0 {
		signer.Issuer = cert.Issuer.Organization[0]
	}

	r.Signer = signer
}

// ComputeValidity sets the Valid field based on individual check results
func (r *VerificationResult) ComputeValidity() {
	r.Valid = r.SignatureFound &&
		r.SignatureValid &&
		r.CertChainValid &&
		r.NotRevoked &&
		len(r.Errors) == 0
}

// IsFullyValid returns true if all checks passed including timestamp
func (r *VerificationResult) IsFullyValid() bool {
	return r.Valid && r.TimestampValid
}
