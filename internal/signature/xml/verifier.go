package xml

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/beevik/etree"
	dsig "github.com/russellhaering/goxmldsig"

	"github.com/rezonia/invoice-processor/internal/signature"
	"github.com/rezonia/invoice-processor/internal/signature/trust"
)

// XMLVerifier verifies XMLDSig signatures
type XMLVerifier struct {
	trustStore *trust.TrustStore
	extractor  *SignatureExtractor
}

// NewXMLVerifier creates a new XML signature verifier
func NewXMLVerifier(ts *trust.TrustStore) *XMLVerifier {
	return &XMLVerifier{
		trustStore: ts,
		extractor:  NewSignatureExtractor(),
	}
}

// Verify verifies the XMLDSig signature in the given XML data
func (v *XMLVerifier) Verify(ctx context.Context, data []byte) (*signature.VerificationResult, error) {
	result := signature.NewVerificationResult()
	result.Format = signature.FormatXML

	// Extract signature element
	extraction, err := v.extractor.Extract(data)
	if err != nil {
		result.AddError(err.Error())
		return result, signature.ErrNoSignature()
	}

	result.SignatureFound = true

	// Verify signature using goxmldsig
	validationCtx := dsig.NewDefaultValidationContext(&dsig.MemoryX509CertificateStore{
		Roots: v.trustStore.RootCerts(),
	})

	// Need to convert etree.Element to something goxmldsig can use
	// goxmldsig works with the signed element, not just the signature
	signedXML, err := elementToBytes(extraction.SignedElement)
	if err != nil {
		result.AddError(fmt.Sprintf("failed to serialize signed element: %v", err))
		return result, err
	}

	// Parse with goxmldsig
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(signedXML); err != nil {
		result.AddError(fmt.Sprintf("failed to parse signed XML: %v", err))
		return result, err
	}

	// Find signature in parsed document
	sigElem := findSignatureElement(doc.Root())
	if sigElem == nil {
		result.AddError("signature element not found after re-parsing")
		return result, signature.ErrNoSignature()
	}

	// Validate the signature
	_, err = validationCtx.Validate(sigElem)
	if err != nil {
		result.SignatureValid = false
		result.AddError(fmt.Sprintf("signature validation failed: %v", err))
		// Don't return early - continue to extract certificate info if possible
	} else {
		result.SignatureValid = true
	}

	// Extract and verify certificate
	cert, certChain, err := v.extractAndVerifyCertificate(ctx, extraction.SignatureElement)
	if err != nil {
		result.AddWarning(fmt.Sprintf("certificate extraction/verification: %v", err))
	} else {
		result.SetSigner(cert)
		result.CertChain = certChain
		result.CertChainValid = true

		// Check revocation
		if len(certChain) >= 2 {
			notRevoked, err := v.trustStore.CheckRevocation(ctx, cert, certChain[1])
			if err != nil {
				if v.trustStore.IsSoftFail() {
					result.AddWarning(fmt.Sprintf("OCSP check: %v (soft-fail enabled)", err))
					result.NotRevoked = true
				} else {
					result.AddError(fmt.Sprintf("OCSP check failed: %v", err))
					result.NotRevoked = false
				}
			} else {
				result.NotRevoked = notRevoked
				if !notRevoked {
					result.AddError("certificate has been revoked")
				}
			}
		} else {
			// Self-signed or no issuer in chain - skip revocation check
			result.NotRevoked = true
			result.AddWarning("revocation check skipped: no issuer certificate in chain")
		}
	}

	// Extract signing time if available
	signingTime := extractSigningTime(extraction.SignatureElement)
	if signingTime != nil {
		result.SignedAt = signingTime
	}

	result.ComputeValidity()
	return result, nil
}

// CanVerify returns true if the data appears to be XML
func (v *XMLVerifier) CanVerify(data []byte) bool {
	// Quick check for XML
	if len(data) < 5 {
		return false
	}

	// Check for XML declaration or root element
	trimmed := bytes.TrimSpace(data)
	return bytes.HasPrefix(trimmed, []byte("<?xml")) || bytes.HasPrefix(trimmed, []byte("<"))
}

// Format returns the format this verifier handles
func (v *XMLVerifier) Format() string {
	return signature.FormatXML
}

// extractAndVerifyCertificate extracts the certificate from signature and verifies chain
func (v *XMLVerifier) extractAndVerifyCertificate(ctx context.Context, sigElem *etree.Element) (*x509.Certificate, []*x509.Certificate, error) {
	// Extract certificate data
	certData, err := ExtractCertificateData(sigElem)
	if err != nil {
		return nil, nil, err
	}

	// Decode base64
	derData, err := base64.StdEncoding.DecodeString(string(bytes.TrimSpace(certData)))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode certificate: %w", err)
	}

	// Parse certificate
	cert, err := x509.ParseCertificate(derData)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Verify chain
	chain, err := v.trustStore.VerifyChain(cert, nil)
	if err != nil {
		return cert, nil, fmt.Errorf("chain verification failed: %w", err)
	}

	return cert, chain, nil
}

// extractSigningTime attempts to extract signing time from the signature
func extractSigningTime(sigElem *etree.Element) *time.Time {
	// Try common paths for signing time
	paths := []string{
		"Object/SignatureProperties/SignatureProperty/SigningTime",
		"Object/SignatureProperties/SigningTime",
		"SignedProperties/SignedSignatureProperties/SigningTime",
	}

	for _, path := range paths {
		if elem := sigElem.FindElement(path); elem != nil {
			if t, err := time.Parse(time.RFC3339, elem.Text()); err == nil {
				return &t
			}
			// Try other formats
			if t, err := time.Parse("2006-01-02T15:04:05", elem.Text()); err == nil {
				return &t
			}
		}
	}

	return nil
}

// elementToBytes converts an etree.Element to bytes
func elementToBytes(elem *etree.Element) ([]byte, error) {
	doc := etree.NewDocument()
	doc.SetRoot(elem.Copy())
	return doc.WriteToBytes()
}
