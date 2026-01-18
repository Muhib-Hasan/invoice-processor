package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rezonia/invoice-processor/internal/model"
	"github.com/rezonia/invoice-processor/internal/processor"
)

var infoCmd = &cobra.Command{
	Use:   "info [files...]",
	Short: "Show information about invoice files",
	Long: `Display information about invoice files without full processing.

Shows:
  - Detected file format (XML, PDF, Image)
  - Detected provider/template (TCT, VNPT, MISA, etc.)
  - File metadata

Examples:
  invoice-processor info invoice.xml
  invoice-processor info *.pdf`,
	Args: cobra.MinimumNArgs(1),
	RunE: runInfo,
}

func init() {
	rootCmd.AddCommand(infoCmd)
}

func runInfo(cmd *cobra.Command, args []string) error {
	files, err := collectFiles(args)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		return fmt.Errorf("no files found")
	}

	for _, file := range files {
		printFileInfo(file)
		fmt.Println()
	}

	return nil
}

func printFileInfo(filePath string) {
	fmt.Printf("File: %s\n", filePath)

	// Get file info
	info, err := os.Stat(filePath)
	if err != nil {
		fmt.Printf("  Error: %v\n", err)
		return
	}

	fmt.Printf("  Size: %d bytes\n", info.Size())
	fmt.Printf("  Modified: %s\n", info.ModTime().Format("2006-01-02 15:04:05"))

	// Read file content
	data, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("  Error reading file: %v\n", err)
		return
	}

	// Detect format
	format := processor.DetectFormat(data)
	fmt.Printf("  Format: %s\n", formatName(format))

	// For XML files, try to detect provider
	if format == processor.FormatXML {
		provider := detectXMLProvider(data)
		fmt.Printf("  Provider: %s\n", providerName(provider))
	}

	// Show preview for text-based files
	if format == processor.FormatXML {
		preview := getPreview(string(data), 200)
		if preview != "" {
			fmt.Printf("  Preview: %s\n", preview)
		}
	}
}

func formatName(f processor.Format) string {
	switch f {
	case processor.FormatXML:
		return "XML (E-Invoice)"
	case processor.FormatPDF:
		return "PDF"
	case processor.FormatImage:
		return "Image"
	default:
		return "Unknown"
	}
}

func detectXMLProvider(data []byte) model.Provider {
	content := string(data)

	// Check for provider-specific markers
	if strings.Contains(content, "<SInvoice>") {
		return model.ProviderVNPT
	}
	if strings.Contains(content, "<HDon>") || strings.Contains(content, "viettel") {
		return model.ProviderViettel
	}
	if strings.Contains(content, "<EInvoice>") || strings.Contains(content, "fpt") {
		return model.ProviderFPT
	}
	if strings.Contains(content, "<MST>") || strings.Contains(content, "<TenHang>") {
		return model.ProviderMISA
	}
	if strings.Contains(content, "<Invoice>") && strings.Contains(content, "<TaxID>") {
		return model.ProviderTCT
	}

	return model.ProviderUnknown
}

func providerName(p model.Provider) string {
	switch p {
	case model.ProviderTCT:
		return "TCT (Tax Authority)"
	case model.ProviderVNPT:
		return "VNPT"
	case model.ProviderMISA:
		return "MISA"
	case model.ProviderViettel:
		return "Viettel"
	case model.ProviderFPT:
		return "FPT"
	default:
		return "Unknown"
	}
}

func getPreview(content string, maxLen int) string {
	// Remove XML declaration
	if idx := strings.Index(content, "?>"); idx >= 0 {
		content = content[idx+2:]
	}

	// Clean up whitespace
	content = strings.TrimSpace(content)
	content = strings.ReplaceAll(content, "\n", " ")
	content = strings.ReplaceAll(content, "\t", " ")

	// Collapse multiple spaces
	for strings.Contains(content, "  ") {
		content = strings.ReplaceAll(content, "  ", " ")
	}

	if len(content) > maxLen {
		content = content[:maxLen] + "..."
	}

	return content
}
