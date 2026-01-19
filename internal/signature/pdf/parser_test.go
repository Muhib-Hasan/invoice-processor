package pdf

import (
	"testing"
	"time"
)

func TestParsePDFSigOutput(t *testing.T) {
	// Sample pdfsig output
	sampleOutput := `Digital Signature Info of: invoice.pdf
Signature #1:
  - Signer Certificate Common Name: CONG TY ABC
  - Signer Certificate Full Distinguished Name: CN=CONG TY ABC,OU=CA Department,O=VNPT-CA,C=VN
  - Signing Time: Jan 15 2025 10:30:00
  - Signing Hash Algorithm: SHA-256
  - Signature Type: adbe.pkcs7.detached
  - Signed Ranges: [0 - 1234], [5678 - 9999]
  - Signature Validation: Signature is Valid.
  - Certificate Validation: Certificate is Trusted.
`

	result, err := ParsePDFSigOutput(sampleOutput)
	if err != nil {
		t.Fatalf("ParsePDFSigOutput failed: %v", err)
	}

	if result.SignatureCount != 1 {
		t.Errorf("SignatureCount: got %d, want 1", result.SignatureCount)
	}

	if len(result.Signatures) != 1 {
		t.Fatalf("Signatures length: got %d, want 1", len(result.Signatures))
	}

	sig := result.Signatures[0]

	if sig.Index != 1 {
		t.Errorf("Index: got %d, want 1", sig.Index)
	}

	if sig.SignerCN != "CONG TY ABC" {
		t.Errorf("SignerCN: got %s, want CONG TY ABC", sig.SignerCN)
	}

	if sig.HashAlgorithm != "SHA-256" {
		t.Errorf("HashAlgorithm: got %s, want SHA-256", sig.HashAlgorithm)
	}

	if sig.SignatureType != "adbe.pkcs7.detached" {
		t.Errorf("SignatureType: got %s, want adbe.pkcs7.detached", sig.SignatureType)
	}

	if !sig.SignatureValid {
		t.Error("SignatureValid should be true")
	}

	if !sig.CertTrusted {
		t.Error("CertTrusted should be true")
	}

	if sig.SigningTime == nil {
		t.Fatal("SigningTime should not be nil")
	}

	expectedTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	if !sig.SigningTime.Equal(expectedTime) {
		t.Errorf("SigningTime: got %v, want %v", sig.SigningTime, expectedTime)
	}
}

func TestParsePDFSigOutput_MultipleSignatures(t *testing.T) {
	sampleOutput := `Digital Signature Info of: invoice.pdf
Signature #1:
  - Signer Certificate Common Name: Signer One
  - Signature Validation: Signature is Valid.
  - Certificate Validation: Certificate is Trusted.
Signature #2:
  - Signer Certificate Common Name: Signer Two
  - Signature Validation: Signature is Valid.
  - Certificate Validation: Certificate is Not Trusted.
`

	result, err := ParsePDFSigOutput(sampleOutput)
	if err != nil {
		t.Fatalf("ParsePDFSigOutput failed: %v", err)
	}

	if result.SignatureCount != 2 {
		t.Errorf("SignatureCount: got %d, want 2", result.SignatureCount)
	}

	if len(result.Signatures) != 2 {
		t.Fatalf("Signatures length: got %d, want 2", len(result.Signatures))
	}

	if result.Signatures[0].SignerCN != "Signer One" {
		t.Errorf("Sig1 SignerCN: got %s", result.Signatures[0].SignerCN)
	}

	if result.Signatures[1].SignerCN != "Signer Two" {
		t.Errorf("Sig2 SignerCN: got %s", result.Signatures[1].SignerCN)
	}

	if !result.Signatures[0].CertTrusted {
		t.Error("Sig1 should be trusted")
	}

	if result.Signatures[1].CertTrusted {
		t.Error("Sig2 should NOT be trusted")
	}
}

func TestParsePDFSigOutput_InvalidSignature(t *testing.T) {
	sampleOutput := `Digital Signature Info of: tampered.pdf
Signature #1:
  - Signer Certificate Common Name: Bad Actor
  - Signature Validation: Signature is Invalid.
  - Certificate Validation: Certificate is Not Trusted.
`

	result, err := ParsePDFSigOutput(sampleOutput)
	if err != nil {
		t.Fatalf("ParsePDFSigOutput failed: %v", err)
	}

	sig := result.Signatures[0]

	if sig.SignatureValid {
		t.Error("SignatureValid should be false")
	}

	if sig.CertTrusted {
		t.Error("CertTrusted should be false")
	}
}

func TestParsePDFSigOutput_NoSignatures(t *testing.T) {
	sampleOutput := `Digital Signature Info of: unsigned.pdf
File 'unsigned.pdf' does not contain any signatures
`

	result, err := ParsePDFSigOutput(sampleOutput)
	if err != nil {
		t.Fatalf("ParsePDFSigOutput failed: %v", err)
	}

	if result.SignatureCount != 0 {
		t.Errorf("SignatureCount: got %d, want 0", result.SignatureCount)
	}
}

func TestParsePDFSigOutput_Empty(t *testing.T) {
	result, err := ParsePDFSigOutput("")
	if err != nil {
		t.Fatalf("ParsePDFSigOutput failed: %v", err)
	}

	if result.SignatureCount != 0 {
		t.Errorf("SignatureCount: got %d, want 0", result.SignatureCount)
	}
}

func TestParseSigningTime(t *testing.T) {
	tests := []struct {
		input    string
		expected *time.Time
	}{
		{
			input:    "Jan 15 2025 10:30:00",
			expected: timePtr(time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)),
		},
		{
			input:    "Jan 2 2025 09:15:30",
			expected: timePtr(time.Date(2025, 1, 2, 9, 15, 30, 0, time.UTC)),
		},
		{
			input:    "invalid date",
			expected: nil,
		},
		{
			input:    "",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseSigningTime(tt.input)
			if tt.expected == nil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
			} else {
				if result == nil {
					t.Errorf("expected %v, got nil", tt.expected)
				} else if !result.Equal(*tt.expected) {
					t.Errorf("got %v, want %v", result, tt.expected)
				}
			}
		})
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}
