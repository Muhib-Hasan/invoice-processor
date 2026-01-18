package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/rezonia/invoice-processor/internal/llm"
	"github.com/rezonia/invoice-processor/internal/model"
	"github.com/rezonia/invoice-processor/internal/processor"
)

var (
	outputFile string
	timeout    time.Duration
)

var processCmd = &cobra.Command{
	Use:   "process [files...]",
	Short: "Process invoice files",
	Long: `Process one or more invoice files and extract structured data.

Supported formats:
  - XML: .xml
  - PDF: .pdf
  - Images: .png, .jpg, .jpeg, .tiff

The extraction flow:
  1. XML files: Direct parsing (fastest, no API key needed)
  2. PDF files: LLM text extraction → LLM vision (requires API key)
  3. Images: LLM vision extraction (requires API key)

Examples:
  invoice-processor process invoice.xml
  invoice-processor process invoice.pdf --api-key <key>
  invoice-processor process *.xml -o results.json
  invoice-processor process invoices/ -f table`,
	Args: cobra.MinimumNArgs(1),
	RunE: runProcess,
}

func init() {
	rootCmd.AddCommand(processCmd)

	processCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file (default: stdout)")
	processCmd.Flags().DurationVar(&timeout, "timeout", 2*time.Minute, "Processing timeout per file")
}

func runProcess(cmd *cobra.Command, args []string) error {
	// Collect all files to process
	files, err := collectFiles(args)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		return fmt.Errorf("no files found to process")
	}

	printVerbose("Found %d files to process\n", len(files))

	// Create pipeline
	var llmExtractor *llm.Extractor
	if apiKey != "" {
		// Build client options
		var clientOpts []llm.ClientOption
		if llmBaseURL != "" {
			clientOpts = append(clientOpts, llm.WithBaseURL(llmBaseURL))
		}

		client := llm.NewClient(apiKey, clientOpts...)

		// Build extractor options
		var extractorOpts []llm.ExtractorOption
		if llmModel != "" {
			extractorOpts = append(extractorOpts, llm.WithTextModel(llmModel))
		}
		if llmVisionModel != "" {
			extractorOpts = append(extractorOpts, llm.WithVisionModel(llmVisionModel))
		}

		llmExtractor = llm.NewExtractor(client, extractorOpts...)
		printVerbose("LLM extraction enabled (text: %s, vision: %s)\n", llmModel, llmVisionModel)
	}

	pipeline := processor.NewPipeline(
		processor.WithLLMExtractor(llmExtractor),
	)

	// Process files
	results := make([]*ProcessResult, 0, len(files))
	for _, file := range files {
		printVerbose("Processing: %s\n", file)

		result := processFile(pipeline, file)
		results = append(results, result)

		if result.Error != "" {
			printVerbose("  Error: %s\n", result.Error)
		} else {
			printVerbose("  Method: %s, Confidence: %.2f\n", result.Method, result.Confidence)
		}
	}

	// Output results
	return outputResults(results)
}

func collectFiles(args []string) ([]string, error) {
	var files []string

	for _, arg := range args {
		// Check if it's a glob pattern
		matches, err := filepath.Glob(arg)
		if err != nil {
			return nil, fmt.Errorf("invalid pattern %s: %w", arg, err)
		}

		if len(matches) == 0 {
			// Check if it's a directory
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
					if !info.IsDir() && isSupportedFile(path) {
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
				if !info.IsDir() && isSupportedFile(match) {
					files = append(files, match)
				}
			}
		}
	}

	return files, nil
}

func isSupportedFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".xml", ".pdf", ".png", ".jpg", ".jpeg", ".tiff", ".tif":
		return true
	default:
		return false
	}
}

func processFile(pipeline *processor.Pipeline, filePath string) *ProcessResult {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	result := &ProcessResult{
		File: filePath,
	}

	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		result.Error = fmt.Sprintf("failed to read file: %v", err)
		return result
	}

	// Detect format
	format := processor.DetectFormat(data)
	ext := strings.ToLower(filepath.Ext(filePath))

	// Override format detection based on extension if ambiguous
	if format == processor.FormatUnknown {
		switch ext {
		case ".xml":
			format = processor.FormatXML
		case ".pdf":
			format = processor.FormatPDF
		case ".png", ".jpg", ".jpeg", ".tiff", ".tif":
			format = processor.FormatImage
		}
	}

	// Process based on format
	var pipelineResult *processor.Result
	switch format {
	case processor.FormatXML:
		pipelineResult = pipeline.ProcessXMLBytes(ctx, data)

	case processor.FormatPDF:
		pipelineResult = pipeline.ProcessPDF(ctx, nil, data, getMimeType(ext))

	case processor.FormatImage:
		pipelineResult = pipeline.ProcessImage(ctx, data, getMimeType(ext))

	default:
		result.Error = "unsupported file format"
		return result
	}

	// Convert result
	if pipelineResult.Error != nil {
		result.Error = pipelineResult.Error.Error()
		return result
	}

	result.Invoice = pipelineResult.Invoice
	result.Method = string(pipelineResult.Method)
	result.Confidence = pipelineResult.Confidence
	result.Warnings = pipelineResult.Warnings

	return result
}

func getMimeType(ext string) string {
	switch ext {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".tiff", ".tif":
		return "image/tiff"
	case ".pdf":
		return "application/pdf"
	default:
		return "application/octet-stream"
	}
}

func outputResults(results []*ProcessResult) error {
	var writer = os.Stdout
	if outputFile != "" {
		f, err := os.Create(outputFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer f.Close()
		writer = f
	}

	switch outputFormat {
	case "json":
		return outputJSON(writer, results)
	case "table":
		return outputTable(writer, results)
	case "csv":
		return outputCSV(writer, results)
	default:
		return fmt.Errorf("unsupported output format: %s", outputFormat)
	}
}

func outputJSON(w *os.File, results []*ProcessResult) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(results)
}

func outputTable(w *os.File, results []*ProcessResult) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "FILE\tNUMBER\tSERIES\tDATE\tTOTAL\tMETHOD\tCONFIDENCE")
	fmt.Fprintln(tw, "----\t------\t------\t----\t-----\t------\t----------")

	for _, r := range results {
		if r.Error != "" {
			fmt.Fprintf(tw, "%s\tERROR: %s\t\t\t\t\t\n", r.File, r.Error)
			continue
		}

		if r.Invoice != nil {
			date := ""
			if !r.Invoice.Date.IsZero() {
				date = r.Invoice.Date.Format("2006-01-02")
			}
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%.2f\n",
				r.File,
				r.Invoice.Number,
				r.Invoice.Series,
				date,
				r.Invoice.TotalAmount.String(),
				r.Method,
				r.Confidence,
			)
		}
	}

	return tw.Flush()
}

func outputCSV(w *os.File, results []*ProcessResult) error {
	fmt.Fprintln(w, "file,number,series,date,seller_name,seller_tax_id,buyer_name,buyer_tax_id,total_amount,currency,method,confidence,error")

	for _, r := range results {
		if r.Error != "" {
			fmt.Fprintf(w, "%s,,,,,,,,,,,,%s\n", r.File, r.Error)
			continue
		}

		if r.Invoice != nil {
			date := ""
			if !r.Invoice.Date.IsZero() {
				date = r.Invoice.Date.Format("2006-01-02")
			}
			fmt.Fprintf(w, "%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%.2f,\n",
				r.File,
				r.Invoice.Number,
				r.Invoice.Series,
				date,
				escapeCSV(r.Invoice.Seller.Name),
				r.Invoice.Seller.TaxID,
				escapeCSV(r.Invoice.Buyer.Name),
				r.Invoice.Buyer.TaxID,
				r.Invoice.TotalAmount.String(),
				r.Invoice.Currency,
				r.Method,
				r.Confidence,
			)
		}
	}

	return nil
}

func escapeCSV(s string) string {
	if strings.Contains(s, ",") || strings.Contains(s, "\"") || strings.Contains(s, "\n") {
		return "\"" + strings.ReplaceAll(s, "\"", "\"\"") + "\""
	}
	return s
}

// ProcessResult holds the result of processing a single file
type ProcessResult struct {
	File       string         `json:"file"`
	Invoice    *model.Invoice `json:"invoice,omitempty"`
	Method     string         `json:"method,omitempty"`
	Confidence float64        `json:"confidence,omitempty"`
	Warnings   []string       `json:"warnings,omitempty"`
	Error      string         `json:"error,omitempty"`
}
