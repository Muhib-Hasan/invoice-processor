package invoicelib

import (
	"context"
	"io"

	"github.com/rezonia/invoice-processor/internal/llm"
	"github.com/rezonia/invoice-processor/internal/model"
	"github.com/rezonia/invoice-processor/internal/processor"
)

// Processor implements Pipeline interface using internal processor
type Processor struct {
	pipeline *processor.Pipeline
	options  PipelineOptions
}

// NewProcessor creates a new invoice processor with the given options
func NewProcessor(opts PipelineOptions) *Processor {
	var llmExtractor *llm.Extractor
	if opts.EnableLLM && opts.LLMAPIKey != "" {
		// Build client options
		var clientOpts []llm.ClientOption
		if opts.LLMBaseURL != "" {
			clientOpts = append(clientOpts, llm.WithBaseURL(opts.LLMBaseURL))
		}

		client := llm.NewClient(opts.LLMAPIKey, clientOpts...)

		// Build extractor options
		var extractorOpts []llm.ExtractorOption
		if opts.LLMModel != "" {
			extractorOpts = append(extractorOpts, llm.WithTextModel(opts.LLMModel))
		}
		if opts.LLMVisionModel != "" {
			extractorOpts = append(extractorOpts, llm.WithVisionModel(opts.LLMVisionModel))
		}

		llmExtractor = llm.NewExtractor(client, extractorOpts...)
	}

	pipeline := processor.NewPipeline(
		processor.WithLLMExtractor(llmExtractor),
	)

	return &Processor{
		pipeline: pipeline,
		options:  opts,
	}
}

// NewDefaultProcessor creates a processor with default options
func NewDefaultProcessor() *Processor {
	return NewProcessor(DefaultPipelineOptions())
}

// Process processes input and returns extraction result
func (p *Processor) Process(ctx context.Context, r io.Reader) (*ExtractionResult, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, &model.ParseError{Message: "failed to read input", Cause: err}
	}

	format := processor.DetectFormat(data)

	var result *processor.Result
	switch format {
	case processor.FormatXML:
		result = p.pipeline.ProcessXMLBytes(ctx, data)
	case processor.FormatPDF:
		result = p.pipeline.ProcessPDF(ctx, nil, data, "application/pdf")
	case processor.FormatImage:
		mimeType := detectMimeType(data)
		result = p.pipeline.ProcessImage(ctx, data, mimeType)
	default:
		return nil, &model.ParseError{Message: "unsupported file format"}
	}

	if result.Error != nil {
		return nil, result.Error
	}

	return &ExtractionResult{
		Invoice:     result.Invoice,
		Confidence:  result.Confidence,
		Method:      string(result.Method),
		Warnings:    result.Warnings,
		NeedsReview: result.Confidence < p.options.ReviewThreshold,
	}, nil
}

// ProcessXML processes XML input directly
func (p *Processor) ProcessXML(ctx context.Context, r io.Reader) (*ExtractionResult, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, &model.ParseError{Message: "failed to read input", Cause: err}
	}

	result := p.pipeline.ProcessXMLBytes(ctx, data)
	if result.Error != nil {
		return nil, result.Error
	}

	return &ExtractionResult{
		Invoice:     result.Invoice,
		Confidence:  result.Confidence,
		Method:      string(result.Method),
		Warnings:    result.Warnings,
		NeedsReview: result.Confidence < p.options.ReviewThreshold,
	}, nil
}

// ProcessPDF processes PDF input directly
func (p *Processor) ProcessPDF(ctx context.Context, r io.Reader) (*ExtractionResult, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, &model.ParseError{Message: "failed to read input", Cause: err}
	}

	result := p.pipeline.ProcessPDF(ctx, nil, data, "application/pdf")
	if result.Error != nil {
		return nil, result.Error
	}

	return &ExtractionResult{
		Invoice:     result.Invoice,
		Confidence:  result.Confidence,
		Method:      string(result.Method),
		Warnings:    result.Warnings,
		NeedsReview: result.Confidence < p.options.ReviewThreshold,
	}, nil
}

// ProcessImage processes image input directly
func (p *Processor) ProcessImage(ctx context.Context, imageData []byte, mimeType string) (*ExtractionResult, error) {
	result := p.pipeline.ProcessImage(ctx, imageData, mimeType)
	if result.Error != nil {
		return nil, result.Error
	}

	return &ExtractionResult{
		Invoice:     result.Invoice,
		Confidence:  result.Confidence,
		Method:      string(result.Method),
		Warnings:    result.Warnings,
		NeedsReview: result.Confidence < p.options.ReviewThreshold,
	}, nil
}

// ProcessBatch processes multiple inputs concurrently
func (p *Processor) ProcessBatch(ctx context.Context, inputs []io.Reader) ([]*ExtractionResult, error) {
	results := make([]*ExtractionResult, len(inputs))
	errCh := make(chan error, len(inputs))

	for i, input := range inputs {
		go func(idx int, r io.Reader) {
			result, err := p.Process(ctx, r)
			if err != nil {
				errCh <- err
				return
			}
			results[idx] = result
			errCh <- nil
		}(i, input)
	}

	// Wait for all goroutines
	var firstErr error
	for range inputs {
		if err := <-errCh; err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return results, firstErr
}

// detectMimeType detects MIME type from file content
func detectMimeType(data []byte) string {
	if len(data) < 8 {
		return "application/octet-stream"
	}

	// PNG
	if data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
		return "image/png"
	}
	// JPEG
	if data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return "image/jpeg"
	}
	// TIFF
	if (data[0] == 0x49 && data[1] == 0x49) || (data[0] == 0x4D && data[1] == 0x4D) {
		return "image/tiff"
	}
	// PDF
	if data[0] == '%' && data[1] == 'P' && data[2] == 'D' && data[3] == 'F' {
		return "application/pdf"
	}
	// XML
	if data[0] == '<' || (data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF && data[3] == '<') {
		return "application/xml"
	}

	return "application/octet-stream"
}
