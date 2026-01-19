package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/rezonia/invoice-processor/internal/signature"
	"github.com/rezonia/invoice-processor/internal/signature/pdf"
	"github.com/rezonia/invoice-processor/internal/signature/trust"
	"github.com/rezonia/invoice-processor/internal/signature/xml"
)

var (
	caFile   string
	skipOCSP bool
)

var verifyCmd = &cobra.Command{
	Use:   "verify [files...]",
	Short: "Verify digital signatures",
	Long: `Verify digital signatures on XML and PDF invoice files.

Verifies:
  - Signature validity (cryptographic verification)
  - Certificate chain (to Vietnam CA roots)
  - Certificate revocation (OCSP, unless --skip-ocsp)
  - Signer information

Supported formats:
  - XML: XMLDSig signatures from Vietnam e-invoice providers
  - PDF: PDF signatures (requires pdfsig tool)

Examples:
  # Verify XML signature
  invoice-processor verify invoice.xml

  # Verify PDF signature
  invoice-processor verify invoice.pdf

  # Verify with custom CA certificate
  invoice-processor verify --ca-file company.crt invoice.xml

  # Skip OCSP revocation check
  invoice-processor verify --skip-ocsp invoice.xml

  # JSON output
  invoice-processor verify -f json invoice.xml`,
	Args: cobra.MinimumNArgs(1),
	RunE: runVerify,
}

func init() {
	rootCmd.AddCommand(verifyCmd)

	verifyCmd.Flags().StringVar(&caFile, "ca-file", "", "Custom CA certificate file (PEM format)")
	verifyCmd.Flags().BoolVar(&skipOCSP, "skip-ocsp", false, "Skip OCSP revocation check")
}

func runVerify(cmd *cobra.Command, args []string) error {
	// Collect files
	files, err := collectVerifyFiles(args)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		return fmt.Errorf("no files found to verify")
	}

	// Create trust store
	var opts []trust.TrustStoreOption
	if caFile != "" {
		opts = append(opts, trust.WithCustomCertsFromFile(caFile))
	}
	if skipOCSP {
		opts = append(opts, trust.WithSoftFail())
	}

	trustStore, err := trust.NewTrustStore(opts...)
	if err != nil {
		return fmt.Errorf("failed to create trust store: %w", err)
	}

	// Create verifiers
	xmlVerifier := xml.NewXMLVerifier(trustStore)
	pdfVerifier := pdf.NewPDFVerifier(trustStore)

	// Create registry
	registry := signature.NewVerifierRegistry()
	registry.Register(xmlVerifier)
	registry.Register(pdfVerifier)

	// Process files
	results := make([]*VerifyResult, 0, len(files))
	allValid := true

	for _, file := range files {
		printVerbose("Verifying: %s\n", file)

		result := verifyFile(registry, pdfVerifier, file)
		results = append(results, result)

		if !result.Valid {
			allValid = false
		}
	}

	// Output results
	if outputFormat == "json" {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(results)
	}

	// Table output
	for _, r := range results {
		statusIcon := "✓"
		statusText := "VALID"
		if !r.Valid {
			statusIcon = "✗"
			statusText = "INVALID"
		}

		fmt.Printf("%s %s: %s\n", statusIcon, r.File, statusText)

		if r.Format != "" {
			fmt.Printf("  Format: %s\n", r.Format)
		}

		if r.Signer != nil {
			fmt.Printf("  Signer: %s\n", r.Signer.Name)
			if r.Signer.Organization != "" {
				fmt.Printf("  Org:    %s\n", r.Signer.Organization)
			}
			if r.Signer.Issuer != "" {
				fmt.Printf("  Issuer: %s\n", r.Signer.Issuer)
			}
		}

		if r.SignedAt != nil {
			fmt.Printf("  Signed: %s\n", r.SignedAt.Format(time.RFC3339))
		}

		// Verification details
		if r.SignatureFound {
			sigStatus := "✓"
			if !r.SignatureValid {
				sigStatus = "✗"
			}
			fmt.Printf("  Signature:  %s\n", sigStatus)

			certStatus := "✓"
			if !r.CertChainValid {
				certStatus = "✗"
			}
			fmt.Printf("  Cert Chain: %s\n", certStatus)

			revokeStatus := "✓"
			if !r.NotRevoked {
				revokeStatus = "✗"
			}
			if skipOCSP {
				revokeStatus = "- (skipped)"
			}
			fmt.Printf("  Not Revoked: %s\n", revokeStatus)
		}

		for _, e := range r.Errors {
			fmt.Printf("  ✗ %s\n", e)
		}
		for _, w := range r.Warnings {
			fmt.Printf("  ⚠ %s\n", w)
		}
	}

	if !allValid {
		return fmt.Errorf("verification failed for some files")
	}

	return nil
}

func verifyFile(registry *signature.VerifierRegistry, pdfVerifier *pdf.PDFVerifier, filePath string) *VerifyResult {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result := &VerifyResult{
		File:     filePath,
		Errors:   []string{},
		Warnings: []string{},
	}

	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("failed to read file: %v", err))
		return result
	}

	// Check if PDF and pdfsig unavailable
	if strings.ToLower(filepath.Ext(filePath)) == ".pdf" && !pdfVerifier.IsAvailable() {
		result.Errors = append(result.Errors, "PDF verification unavailable: pdfsig not installed")
		result.Warnings = append(result.Warnings, pdf.GetInstallInstructions())
		return result
	}

	// Find appropriate verifier
	verifier, err := registry.Detect(data)
	if err != nil {
		result.Errors = append(result.Errors, "no verifier available for this file format")
		return result
	}

	result.Format = verifier.Format()

	// Verify
	verifyResult, err := verifier.Verify(ctx, data)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("verification error: %v", err))
		return result
	}

	// Map result
	result.Valid = verifyResult.Valid
	result.SignatureFound = verifyResult.SignatureFound
	result.SignatureValid = verifyResult.SignatureValid
	result.CertChainValid = verifyResult.CertChainValid
	result.NotRevoked = verifyResult.NotRevoked
	result.SignedAt = verifyResult.SignedAt
	result.Errors = append(result.Errors, verifyResult.Errors...)
	result.Warnings = append(result.Warnings, verifyResult.Warnings...)

	if verifyResult.Signer != nil {
		result.Signer = &SignerOutput{
			Name:         verifyResult.Signer.Name,
			Organization: verifyResult.Signer.Organization,
			SerialNumber: verifyResult.Signer.SerialNumber,
			Issuer:       verifyResult.Signer.Issuer,
			ValidFrom:    &verifyResult.Signer.ValidFrom,
			ValidTo:      &verifyResult.Signer.ValidTo,
		}
	}

	return result
}

// collectVerifyFiles collects files for verification
func collectVerifyFiles(args []string) ([]string, error) {
	var files []string

	for _, arg := range args {
		// Check if it's a glob pattern
		matches, err := filepath.Glob(arg)
		if err != nil {
			return nil, fmt.Errorf("invalid pattern %s: %w", arg, err)
		}

		if len(matches) == 0 {
			// Check if it exists
			info, err := os.Stat(arg)
			if err != nil {
				return nil, fmt.Errorf("file not found: %s", arg)
			}

			if info.IsDir() {
				// Walk directory
				err := filepath.Walk(arg, func(path string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}
					if !info.IsDir() && isVerifiableFile(path) {
						files = append(files, path)
					}
					return nil
				})
				if err != nil {
					return nil, err
				}
			} else {
				files = append(files, arg)
			}
		} else {
			for _, match := range matches {
				info, err := os.Stat(match)
				if err != nil {
					continue
				}
				if !info.IsDir() && isVerifiableFile(match) {
					files = append(files, match)
				}
			}
		}
	}

	return files, nil
}

func isVerifiableFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".xml", ".pdf":
		return true
	default:
		return false
	}
}

// VerifyResult holds the result of verifying a single file
type VerifyResult struct {
	File           string        `json:"file"`
	Valid          bool          `json:"valid"`
	Format         string        `json:"format,omitempty"`
	SignatureFound bool          `json:"signature_found"`
	SignatureValid bool          `json:"signature_valid"`
	CertChainValid bool          `json:"cert_chain_valid"`
	NotRevoked     bool          `json:"not_revoked"`
	Signer         *SignerOutput `json:"signer,omitempty"`
	SignedAt       *time.Time    `json:"signed_at,omitempty"`
	Errors         []string      `json:"errors,omitempty"`
	Warnings       []string      `json:"warnings,omitempty"`
}

// SignerOutput holds signer info for output
type SignerOutput struct {
	Name         string     `json:"name,omitempty"`
	Organization string     `json:"organization,omitempty"`
	SerialNumber string     `json:"serial_number,omitempty"`
	Issuer       string     `json:"issuer,omitempty"`
	ValidFrom    *time.Time `json:"valid_from,omitempty"`
	ValidTo      *time.Time `json:"valid_to,omitempty"`
}
