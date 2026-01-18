package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	version = "1.0.0"

	// Global flags
	verbose        bool
	outputFormat   string
	apiKey         string
	llmBaseURL     string
	llmModel       string
	llmVisionModel string
)

var rootCmd = &cobra.Command{
	Use:   "invoice-processor",
	Short: "Process Vietnam e-invoices (XML and PDF)",
	Long: `Invoice Processor is a CLI tool for extracting data from Vietnam e-invoices.

Supports:
  - XML formats: TCT, VNPT, MISA, Viettel, FPT
  - PDF invoices: template matching and LLM-based extraction
  - Image invoices: LLM vision extraction

Examples:
  # Process a single XML file
  invoice-processor process invoice.xml

  # Process a PDF with LLM fallback
  invoice-processor process invoice.pdf --api-key <openrouter-key>

  # Process multiple files
  invoice-processor process *.xml *.pdf -o results.json

  # Validate an invoice
  invoice-processor validate invoice.xml`,
	Version: version,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "format", "f", "json", "Output format (json, csv, table)")
	rootCmd.PersistentFlags().StringVar(&apiKey, "api-key", "", "API key for LLM provider (env: LLM_API_KEY)")
	rootCmd.PersistentFlags().StringVar(&llmBaseURL, "llm-base-url", "", "LLM API base URL (env: LLM_BASE_URL)")
	rootCmd.PersistentFlags().StringVar(&llmModel, "llm-model", "", "LLM model for text extraction (env: LLM_MODEL)")
	rootCmd.PersistentFlags().StringVar(&llmVisionModel, "llm-vision-model", "", "LLM model for vision/image extraction (env: LLM_VISION_MODEL)")

	// Load from environment variables if not set via flags
	cobra.OnInitialize(initConfig)
}

func initConfig() {
	// API key
	if apiKey == "" {
		apiKey = os.Getenv("LLM_API_KEY")
	}
	// Base URL
	if llmBaseURL == "" {
		llmBaseURL = os.Getenv("LLM_BASE_URL")
	}
	// Text model
	if llmModel == "" {
		llmModel = os.Getenv("LLM_MODEL")
	}
	// Vision model
	if llmVisionModel == "" {
		llmVisionModel = os.Getenv("LLM_VISION_MODEL")
	}
}

func printVerbose(format string, args ...interface{}) {
	if verbose {
		fmt.Fprintf(os.Stderr, format, args...)
	}
}
