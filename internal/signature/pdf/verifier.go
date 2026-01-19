package pdf

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/rezonia/invoice-processor/internal/signature"
	"github.com/rezonia/invoice-processor/internal/signature/trust"
)

// PDFVerifier verifies PDF signatures using external pdfsig tool
type PDFVerifier struct {
	trustStore *trust.TrustStore
	pdfsigPath string
	available  bool
	timeout    time.Duration
}

// PDF magic bytes
var pdfMagic = []byte("%PDF")

// NewPDFVerifier creates a new PDF signature verifier
func NewPDFVerifier(ts *trust.TrustStore) *PDFVerifier {
	path, available := detectPDFSig()
	return &PDFVerifier{
		trustStore: ts,
		pdfsigPath: path,
		available:  available,
		timeout:    30 * time.Second,
	}
}

// Verify verifies the PDF signature(s) in the given PDF data
func (v *PDFVerifier) Verify(ctx context.Context, data []byte) (*signature.VerificationResult, error) {
	result := signature.NewVerificationResult()
	result.Format = signature.FormatPDF

	if !v.available {
		result.AddError("pdfsig tool not available")
		return result, signature.ErrToolUnavailable("pdfsig")
	}

	// Write PDF to temp file (pdfsig requires a file path)
	tmpFile, err := os.CreateTemp("", "verify-*.pdf")
	if err != nil {
		result.AddError(fmt.Sprintf("failed to create temp file: %v", err))
		return result, err
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := tmpFile.Write(data); err != nil {
		result.AddError(fmt.Sprintf("failed to write temp file: %v", err))
		return result, err
	}
	tmpFile.Close()

	// Run pdfsig
	ctx, cancel := context.WithTimeout(ctx, v.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, v.pdfsigPath, "-dump", tmpFile.Name())
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		// pdfsig may return non-zero even with valid output
		// Check if we got any output
		if stdout.Len() == 0 {
			result.AddError(fmt.Sprintf("pdfsig failed: %v, stderr: %s", err, stderr.String()))
			return result, err
		}
	}

	// Parse output
	output, err := ParsePDFSigOutput(stdout.String())
	if err != nil {
		result.AddError(fmt.Sprintf("failed to parse pdfsig output: %v", err))
		return result, err
	}

	if output.SignatureCount == 0 {
		result.AddError("no signatures found in PDF")
		return result, signature.ErrNoSignature()
	}

	result.SignatureFound = true

	// Process first signature (primary)
	// In practice, most PDFs have one signature
	if len(output.Signatures) > 0 {
		sig := output.Signatures[0]

		result.SignatureValid = sig.SignatureValid
		result.CertChainValid = sig.CertTrusted

		if sig.SigningTime != nil {
			result.SignedAt = sig.SigningTime
		}

		// Set signer info from parsed output
		if sig.SignerCN != "" || sig.SignerDN != "" {
			result.Signer = &signature.SignerInfo{
				Name:   sig.SignerCN,
				Issuer: extractIssuerFromDN(sig.SignerDN),
			}
		}

		if !sig.SignatureValid {
			result.AddError(fmt.Sprintf("signature invalid: %s", sig.ErrorMessage))
		}

		if !sig.CertTrusted {
			result.AddWarning("certificate not trusted by pdfsig")
		}
	}

	// Add warnings for additional signatures
	if output.SignatureCount > 1 {
		result.AddWarning(fmt.Sprintf("PDF contains %d signatures, only first verified", output.SignatureCount))
	}

	// pdfsig doesn't do OCSP, so we mark as not checked
	result.NotRevoked = true
	result.AddWarning("OCSP revocation check not performed for PDF signatures")

	result.ComputeValidity()
	return result, nil
}

// CanVerify returns true if the data appears to be a PDF
func (v *PDFVerifier) CanVerify(data []byte) bool {
	return bytes.HasPrefix(data, pdfMagic)
}

// Format returns the format this verifier handles
func (v *PDFVerifier) Format() string {
	return signature.FormatPDF
}

// IsAvailable returns whether pdfsig tool is available
func (v *PDFVerifier) IsAvailable() bool {
	return v.available
}

// SetTimeout sets the execution timeout for pdfsig
func (v *PDFVerifier) SetTimeout(d time.Duration) {
	v.timeout = d
}

// detectPDFSig looks for pdfsig in common locations
func detectPDFSig() (string, bool) {
	// Check common paths
	paths := []string{
		"pdfsig",                   // PATH
		"/usr/bin/pdfsig",          // Linux
		"/opt/homebrew/bin/pdfsig", // macOS Homebrew ARM
		"/usr/local/bin/pdfsig",    // macOS Homebrew Intel
	}

	for _, p := range paths {
		if path, err := exec.LookPath(p); err == nil {
			return path, true
		}
	}

	// Try to find in PATH
	if path, err := exec.LookPath("pdfsig"); err == nil {
		return path, true
	}

	return "", false
}

// extractIssuerFromDN extracts the issuer organization from a distinguished name
func extractIssuerFromDN(dn string) string {
	// Parse DN like "CN=...,OU=...,O=Issuer Name,C=VN"
	parts := bytes.Split([]byte(dn), []byte(","))
	for _, part := range parts {
		part = bytes.TrimSpace(part)
		if bytes.HasPrefix(part, []byte("O=")) {
			return string(part[2:])
		}
	}
	return ""
}

// GetInstallInstructions returns platform-specific installation instructions
func GetInstallInstructions() string {
	return `pdfsig is required for PDF signature verification.

Installation:
  - Ubuntu/Debian: sudo apt install poppler-utils
  - macOS:         brew install poppler
  - Fedora/RHEL:   sudo dnf install poppler-utils
  - Windows:       Install poppler from https://github.com/oschwartz10612/poppler-windows/releases

After installation, ensure 'pdfsig' is in your PATH.`
}

// GetPDFSigPath returns the detected pdfsig path (for testing/debugging)
func (v *PDFVerifier) GetPDFSigPath() string {
	return v.pdfsigPath
}

// CreateUnavailableResult creates a result for when PDF verification is unavailable
func CreateUnavailableResult() *signature.VerificationResult {
	result := signature.NewVerificationResult()
	result.Format = signature.FormatPDF
	result.AddError("PDF signature verification unavailable: pdfsig tool not installed")
	result.AddWarning(GetInstallInstructions())
	return result
}
