package signature

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"math/big"
	"testing"
	"time"
)

func TestVerificationResult_JSONSerialization(t *testing.T) {
	signedAt := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	result := &VerificationResult{
		Valid:          true,
		SignatureFound: true,
		SignatureValid: true,
		CertChainValid: true,
		NotRevoked:     true,
		TimestampValid: true,
		SignedAt:       &signedAt,
		Format:         FormatXML,
		Signer: &SignerInfo{
			Name:         "CÔNG TY ABC",
			Organization: "ABC Company Ltd",
			SerialNumber: "1234567890",
			Issuer:       "VNPT-CA",
			ValidFrom:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			ValidTo:      time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC),
		},
		Warnings: []string{"OCSP response cached"},
		Errors:   []string{},
	}

	// Marshal to JSON
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal result: %v", err)
	}

	// Unmarshal back
	var decoded VerificationResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	// Verify fields
	if decoded.Valid != result.Valid {
		t.Errorf("Valid: got %v, want %v", decoded.Valid, result.Valid)
	}
	if decoded.SignatureFound != result.SignatureFound {
		t.Errorf("SignatureFound: got %v, want %v", decoded.SignatureFound, result.SignatureFound)
	}
	if decoded.SignatureValid != result.SignatureValid {
		t.Errorf("SignatureValid: got %v, want %v", decoded.SignatureValid, result.SignatureValid)
	}
	if decoded.Format != result.Format {
		t.Errorf("Format: got %v, want %v", decoded.Format, result.Format)
	}
	if decoded.Signer == nil {
		t.Fatal("Signer is nil after unmarshal")
	}
	if decoded.Signer.Name != result.Signer.Name {
		t.Errorf("Signer.Name: got %v, want %v", decoded.Signer.Name, result.Signer.Name)
	}
	if decoded.Signer.Issuer != result.Signer.Issuer {
		t.Errorf("Signer.Issuer: got %v, want %v", decoded.Signer.Issuer, result.Signer.Issuer)
	}
	if len(decoded.Warnings) != len(result.Warnings) {
		t.Errorf("Warnings length: got %d, want %d", len(decoded.Warnings), len(result.Warnings))
	}
}

func TestVerificationResult_OmitEmpty(t *testing.T) {
	result := &VerificationResult{
		Valid:          false,
		SignatureFound: false,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// Check that empty fields are omitted
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	if _, exists := raw["signer"]; exists {
		t.Error("signer should be omitted when nil")
	}
	if _, exists := raw["signed_at"]; exists {
		t.Error("signed_at should be omitted when nil")
	}
	if _, exists := raw["timestamp_valid"]; exists {
		t.Error("timestamp_valid should be omitted when false (default)")
	}
}

func TestVerificationResult_CertChainNotInJSON(t *testing.T) {
	// Create a dummy certificate for testing
	key, _ := rsa.GenerateKey(rand.Reader, 1024)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "Test Cert",
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(time.Hour),
	}
	certDER, _ := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	cert, _ := x509.ParseCertificate(certDER)

	result := &VerificationResult{
		Valid:     true,
		CertChain: []*x509.Certificate{cert},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// CertChain should not appear in JSON
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	if _, exists := raw["cert_chain"]; exists {
		t.Error("cert_chain should not be serialized to JSON")
	}
}

func TestVerificationResult_SetSigner(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 1024)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(12345),
		Subject: pkix.Name{
			CommonName:   "CÔNG TY XYZ",
			Organization: []string{"XYZ Corp"},
		},
		Issuer: pkix.Name{
			CommonName:   "Test CA",
			Organization: []string{"Test CA Org"},
		},
		NotBefore: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		NotAfter:  time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
	}
	certDER, _ := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	cert, _ := x509.ParseCertificate(certDER)

	result := NewVerificationResult()
	result.SetSigner(cert)

	if result.Signer == nil {
		t.Fatal("Signer is nil after SetSigner")
	}
	if result.Signer.Name != "CÔNG TY XYZ" {
		t.Errorf("Name: got %v, want CÔNG TY XYZ", result.Signer.Name)
	}
	if result.Signer.Organization != "XYZ Corp" {
		t.Errorf("Organization: got %v, want XYZ Corp", result.Signer.Organization)
	}
	if result.Signer.SerialNumber != "12345" {
		t.Errorf("SerialNumber: got %v, want 12345", result.Signer.SerialNumber)
	}
}

func TestVerificationResult_ComputeValidity(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*VerificationResult)
		expected bool
	}{
		{
			name: "all checks pass",
			setup: func(r *VerificationResult) {
				r.SignatureFound = true
				r.SignatureValid = true
				r.CertChainValid = true
				r.NotRevoked = true
			},
			expected: true,
		},
		{
			name: "signature not found",
			setup: func(r *VerificationResult) {
				r.SignatureFound = false
				r.SignatureValid = true
				r.CertChainValid = true
				r.NotRevoked = true
			},
			expected: false,
		},
		{
			name: "signature invalid",
			setup: func(r *VerificationResult) {
				r.SignatureFound = true
				r.SignatureValid = false
				r.CertChainValid = true
				r.NotRevoked = true
			},
			expected: false,
		},
		{
			name: "chain invalid",
			setup: func(r *VerificationResult) {
				r.SignatureFound = true
				r.SignatureValid = true
				r.CertChainValid = false
				r.NotRevoked = true
			},
			expected: false,
		},
		{
			name: "certificate revoked",
			setup: func(r *VerificationResult) {
				r.SignatureFound = true
				r.SignatureValid = true
				r.CertChainValid = true
				r.NotRevoked = false
			},
			expected: false,
		},
		{
			name: "has errors",
			setup: func(r *VerificationResult) {
				r.SignatureFound = true
				r.SignatureValid = true
				r.CertChainValid = true
				r.NotRevoked = true
				r.Errors = []string{"some error"}
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewVerificationResult()
			tt.setup(result)
			result.ComputeValidity()

			if result.Valid != tt.expected {
				t.Errorf("Valid: got %v, want %v", result.Valid, tt.expected)
			}
		})
	}
}

func TestVerificationResult_AddWarningAndError(t *testing.T) {
	result := NewVerificationResult()
	result.Valid = true

	result.AddWarning("OCSP cache hit")
	if len(result.Warnings) != 1 {
		t.Errorf("Warnings count: got %d, want 1", len(result.Warnings))
	}
	if result.Valid != true {
		t.Error("AddWarning should not change Valid")
	}

	result.AddError("Certificate expired")
	if len(result.Errors) != 1 {
		t.Errorf("Errors count: got %d, want 1", len(result.Errors))
	}
	if result.Valid != false {
		t.Error("AddError should set Valid to false")
	}
}
