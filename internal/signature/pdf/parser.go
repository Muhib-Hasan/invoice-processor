package pdf

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// PDFSigOutput represents parsed pdfsig output
type PDFSigOutput struct {
	SignatureCount int
	Signatures     []PDFSignature
}

// PDFSignature represents a single signature from pdfsig output
type PDFSignature struct {
	Index          int
	SignerCN       string
	SignerDN       string
	SigningTime    *time.Time
	HashAlgorithm  string
	SignatureType  string
	SignatureValid bool
	CertTrusted    bool
	ErrorMessage   string
}

// Regex patterns for parsing pdfsig output
var (
	sigIndexPattern    = regexp.MustCompile(`Signature #(\d+):`)
	signerCNPattern    = regexp.MustCompile(`Signer Certificate Common Name:\s*(.+)`)
	signerDNPattern    = regexp.MustCompile(`Signer Certificate Full Distinguished Name:\s*(.+)`)
	signingTimePattern = regexp.MustCompile(`Signing Time:\s*(.+)`)
	hashAlgoPattern    = regexp.MustCompile(`Signing Hash Algorithm:\s*(.+)`)
	sigTypePattern     = regexp.MustCompile(`Signature Type:\s*(.+)`)
	sigValidPattern    = regexp.MustCompile(`Signature Validation:\s*(.+)`)
	certTrustedPattern = regexp.MustCompile(`Certificate Validation:\s*(.+)`)
)

// ParsePDFSigOutput parses the text output from pdfsig command
func ParsePDFSigOutput(output string) (*PDFSigOutput, error) {
	result := &PDFSigOutput{
		Signatures: make([]PDFSignature, 0),
	}

	// Split into lines
	lines := strings.Split(output, "\n")

	var currentSig *PDFSignature

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check for new signature
		if matches := sigIndexPattern.FindStringSubmatch(line); len(matches) > 1 {
			// Save previous signature
			if currentSig != nil {
				result.Signatures = append(result.Signatures, *currentSig)
			}
			// Start new signature
			idx, _ := strconv.Atoi(matches[1])
			currentSig = &PDFSignature{
				Index: idx,
			}
			result.SignatureCount++
			continue
		}

		if currentSig == nil {
			continue
		}

		// Parse signature fields
		if matches := signerCNPattern.FindStringSubmatch(line); len(matches) > 1 {
			currentSig.SignerCN = strings.TrimSpace(matches[1])
			continue
		}

		if matches := signerDNPattern.FindStringSubmatch(line); len(matches) > 1 {
			currentSig.SignerDN = strings.TrimSpace(matches[1])
			continue
		}

		if matches := signingTimePattern.FindStringSubmatch(line); len(matches) > 1 {
			if t := parseSigningTime(matches[1]); t != nil {
				currentSig.SigningTime = t
			}
			continue
		}

		if matches := hashAlgoPattern.FindStringSubmatch(line); len(matches) > 1 {
			currentSig.HashAlgorithm = strings.TrimSpace(matches[1])
			continue
		}

		if matches := sigTypePattern.FindStringSubmatch(line); len(matches) > 1 {
			currentSig.SignatureType = strings.TrimSpace(matches[1])
			continue
		}

		if matches := sigValidPattern.FindStringSubmatch(line); len(matches) > 1 {
			status := strings.TrimSpace(matches[1])
			currentSig.SignatureValid = strings.Contains(strings.ToLower(status), "valid") &&
				!strings.Contains(strings.ToLower(status), "invalid")
			if !currentSig.SignatureValid {
				currentSig.ErrorMessage = status
			}
			continue
		}

		if matches := certTrustedPattern.FindStringSubmatch(line); len(matches) > 1 {
			status := strings.TrimSpace(matches[1])
			currentSig.CertTrusted = strings.Contains(strings.ToLower(status), "trusted") &&
				!strings.Contains(strings.ToLower(status), "not trusted")
			continue
		}
	}

	// Save last signature
	if currentSig != nil {
		result.Signatures = append(result.Signatures, *currentSig)
	}

	return result, nil
}

// parseSigningTime attempts to parse various date formats from pdfsig
func parseSigningTime(s string) *time.Time {
	s = strings.TrimSpace(s)

	// Common formats from pdfsig
	formats := []string{
		"Jan 02 2006 15:04:05",
		"Jan 2 2006 15:04:05",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05-07:00",
		"Mon Jan 2 15:04:05 2006",
		"Mon Jan 02 15:04:05 2006",
		time.RFC3339,
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return &t
		}
	}

	return nil
}
