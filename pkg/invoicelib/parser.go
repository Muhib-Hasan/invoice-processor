package invoicelib

import (
	"context"
	"io"

	"github.com/rezonia/invoice-processor/internal/model"
)

// Parser parses invoices from various formats
type Parser interface {
	// ParseXML parses XML content into Invoice
	ParseXML(ctx context.Context, r io.Reader) (*model.Invoice, error)

	// ParsePDF extracts invoice data from PDF
	ParsePDF(ctx context.Context, r io.Reader) (*model.Invoice, float64, error)

	// Parse auto-detects format and parses
	Parse(ctx context.Context, r io.Reader) (*model.Invoice, error)

	// DetectProvider identifies provider from content
	DetectProvider(content []byte) (model.Provider, error)
}

// Validator validates invoice data
type Validator interface {
	// Validate performs full validation
	Validate(inv *model.Invoice) []ValidationResult

	// ValidateTaxID validates Vietnam tax ID format
	ValidateTaxID(taxID string) error

	// ValidateInvoiceNumber validates invoice number format
	ValidateInvoiceNumber(number string) error

	// ValidateSeries validates invoice series format
	ValidateSeries(series string) error
}

// ValidationResult represents a validation result
type ValidationResult struct {
	Field   string
	Message string
	Value   interface{}
	IsError bool // true = error, false = warning
}

// Extractor extracts structured data from unstructured sources
type Extractor interface {
	// ExtractFromText extracts invoice data from OCR text using LLM
	ExtractFromText(ctx context.Context, text string) (*model.Invoice, float64, error)

	// ExtractFromImage extracts invoice data from image/PDF using LLM vision
	ExtractFromImage(ctx context.Context, imageData []byte, mimeType string) (*model.Invoice, float64, error)
}

// ExtractionResult represents extraction result with metadata
type ExtractionResult struct {
	Invoice     *model.Invoice
	Confidence  float64
	Method      string
	Warnings    []string
	NeedsReview bool
}

// Pipeline processes invoices through the extraction chain
type Pipeline interface {
	// Process processes input and returns extraction result
	Process(ctx context.Context, r io.Reader) (*ExtractionResult, error)

	// ProcessBatch processes multiple inputs
	ProcessBatch(ctx context.Context, inputs []io.Reader) ([]*ExtractionResult, error)
}

// PipelineOptions configures pipeline behavior
type PipelineOptions struct {
	// Thresholds
	TemplateThreshold float64 // Minimum confidence for template (default: 0.90)
	LLMThreshold      float64 // Minimum confidence for LLM (default: 0.85)
	ReviewThreshold   float64 // Below this, flag for review (default: 0.70)

	// LLM Configuration
	LLMAPIKey      string // API key (env: LLM_API_KEY)
	LLMBaseURL     string // Base URL (env: LLM_BASE_URL)
	LLMModel       string // Text extraction model (env: LLM_MODEL)
	LLMVisionModel string // Vision/image extraction model (env: LLM_VISION_MODEL)

	// Feature flags
	EnableLLM bool
	EnableOCR bool

	// Validation
	ValidateAfterExtraction bool
}

// DefaultPipelineOptions returns default pipeline options
func DefaultPipelineOptions() PipelineOptions {
	return PipelineOptions{
		TemplateThreshold:       0.90,
		LLMThreshold:            0.85,
		ReviewThreshold:         0.70,
		EnableLLM:               true,
		EnableOCR:               true,
		ValidateAfterExtraction: true,
		LLMBaseURL:              "https://openrouter.ai/api/v1",
		LLMModel:                "anthropic/claude-3.5-sonnet",
		LLMVisionModel:          "anthropic/claude-3.5-sonnet",
	}
}
