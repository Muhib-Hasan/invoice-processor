package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/rezonia/invoice-processor/internal/processor"
)

var (
	strictValidation bool
)

var validateCmd = &cobra.Command{
	Use:   "validate [files...]",
	Short: "Validate invoice files",
	Long: `Validate one or more invoice files for completeness and correctness.

Checks performed:
  - Required fields present (number, date, seller, buyer)
  - Tax ID format (10 or 13 digits for Vietnam)
  - Amount calculations (subtotal + tax = total)
  - Date validity

Examples:
  invoice-processor validate invoice.xml
  invoice-processor validate *.xml --strict`,
	Args: cobra.MinimumNArgs(1),
	RunE: runValidate,
}

func init() {
	rootCmd.AddCommand(validateCmd)

	validateCmd.Flags().BoolVar(&strictValidation, "strict", false, "Enable strict validation (all fields required)")
}

func runValidate(cmd *cobra.Command, args []string) error {
	files, err := collectFiles(args)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		return fmt.Errorf("no files found to validate")
	}

	pipeline := processor.NewPipeline()
	results := make([]*ValidationResult, 0, len(files))
	allValid := true

	for _, file := range files {
		result := validateFile(pipeline, file)
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
		if r.Valid {
			fmt.Printf("✓ %s: VALID\n", r.File)
		} else {
			fmt.Printf("✗ %s: INVALID\n", r.File)
			for _, e := range r.Errors {
				fmt.Printf("  - %s\n", e)
			}
		}
		for _, w := range r.Warnings {
			fmt.Printf("  ⚠ %s\n", w)
		}
	}

	if !allValid {
		return fmt.Errorf("validation failed for some files")
	}

	return nil
}

func validateFile(pipeline *processor.Pipeline, filePath string) *ValidationResult {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result := &ValidationResult{
		File:     filePath,
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}

	// Read and process file
	data, err := os.ReadFile(filePath)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("failed to read file: %v", err))
		return result
	}

	// Only validate XML files for now
	format := processor.DetectFormat(data)
	if format != processor.FormatXML {
		result.Warnings = append(result.Warnings, "only XML validation is fully supported")
	}

	// Process file
	var pipelineResult *processor.Result
	switch format {
	case processor.FormatXML:
		pipelineResult = pipeline.ProcessXMLBytes(ctx, data)
	default:
		result.Warnings = append(result.Warnings, "skipping validation for non-XML file")
		return result
	}

	if pipelineResult.Error != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("parse error: %v", pipelineResult.Error))
		return result
	}

	// Validate invoice
	inv := pipelineResult.Invoice
	if inv == nil {
		result.Valid = false
		result.Errors = append(result.Errors, "no invoice data extracted")
		return result
	}

	// Required field validation
	if inv.Number == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "missing invoice number")
	}

	if inv.Date.IsZero() {
		if strictValidation {
			result.Valid = false
			result.Errors = append(result.Errors, "missing invoice date")
		} else {
			result.Warnings = append(result.Warnings, "missing invoice date")
		}
	}

	// Seller validation
	if inv.Seller.TaxID == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "missing seller tax ID")
	} else if !isValidTaxID(inv.Seller.TaxID) {
		result.Warnings = append(result.Warnings, fmt.Sprintf("seller tax ID format may be invalid: %s", inv.Seller.TaxID))
	}

	if inv.Seller.Name == "" && strictValidation {
		result.Valid = false
		result.Errors = append(result.Errors, "missing seller name")
	}

	// Buyer validation
	if strictValidation {
		if inv.Buyer.TaxID == "" {
			result.Valid = false
			result.Errors = append(result.Errors, "missing buyer tax ID")
		}
		if inv.Buyer.Name == "" {
			result.Valid = false
			result.Errors = append(result.Errors, "missing buyer name")
		}
	}

	// Amount validation
	if inv.TotalAmount.IsZero() {
		result.Warnings = append(result.Warnings, "total amount is zero or missing")
	}

	// Check calculation: subtotal + tax = total
	if !inv.SubtotalAmount.IsZero() && !inv.TaxAmount.IsZero() && !inv.TotalAmount.IsZero() {
		expected := inv.SubtotalAmount.Add(inv.TaxAmount)
		if !expected.Equal(inv.TotalAmount) {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("amount mismatch: subtotal(%s) + tax(%s) = %s, but total is %s",
					inv.SubtotalAmount, inv.TaxAmount, expected, inv.TotalAmount))
		}
	}

	// Line item validation
	for i, item := range inv.Items {
		if item.Name == "" {
			result.Warnings = append(result.Warnings, fmt.Sprintf("line item %d: missing name", i+1))
		}
		if item.Quantity.IsZero() {
			result.Warnings = append(result.Warnings, fmt.Sprintf("line item %d: quantity is zero", i+1))
		}
	}

	return result
}

func isValidTaxID(taxID string) bool {
	// Vietnam tax ID: 10 or 13 digits
	if len(taxID) != 10 && len(taxID) != 13 {
		return false
	}

	for _, c := range taxID {
		if c < '0' || c > '9' {
			// Allow hyphens for some formats
			if c != '-' {
				return false
			}
		}
	}

	return true
}

// ValidationResult holds the result of validating a single file
type ValidationResult struct {
	File     string   `json:"file"`
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}
