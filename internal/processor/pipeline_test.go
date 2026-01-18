package processor_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rezonia/invoice-processor/internal/processor"
)

func TestNewPipeline(t *testing.T) {
	p := processor.NewPipeline()
	require.NotNil(t, p)
}

func TestNewPipeline_WithOptions(t *testing.T) {
	p := processor.NewPipeline(
		processor.WithLLMExtractor(nil),
	)
	require.NotNil(t, p)
}

func TestProcessXML_TCT(t *testing.T) {
	ctx := context.Background()
	p := processor.NewPipeline()

	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<Invoice>
	<InvoiceNo>0000001</InvoiceNo>
	<InvoiceSeries>AA/23E</InvoiceSeries>
	<InvoiceDate>2026-01-15</InvoiceDate>
	<Seller>
		<TaxID>0123456789</TaxID>
		<Name>ABC Company</Name>
	</Seller>
	<Buyer>
		<TaxID>9876543210</TaxID>
		<Name>XYZ Corp</Name>
	</Buyer>
	<TotalAmount>1100000</TotalAmount>
	<TaxAmount>100000</TaxAmount>
</Invoice>`

	result := p.ProcessXML(ctx, strings.NewReader(xmlData))
	require.Nil(t, result.Error)
	require.NotNil(t, result.Invoice)

	assert.Equal(t, processor.MethodXML, result.Method)
	assert.Equal(t, 1.0, result.Confidence)
	assert.Equal(t, "0000001", result.Invoice.Number)
	assert.Equal(t, "AA/23E", result.Invoice.Series)
	assert.Equal(t, "0123456789", result.Invoice.Seller.TaxID)
	assert.Equal(t, "ABC Company", result.Invoice.Seller.Name)
}

func TestProcessXML_Invalid(t *testing.T) {
	ctx := context.Background()
	p := processor.NewPipeline()

	result := p.ProcessXML(ctx, strings.NewReader("not xml"))
	require.NotNil(t, result.Error)
	assert.Contains(t, result.Error.Error(), "XML parsing failed")
}

func TestProcessXMLBytes(t *testing.T) {
	ctx := context.Background()
	p := processor.NewPipeline()

	xmlData := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<Invoice>
	<InvoiceNo>0000002</InvoiceNo>
	<Seller><TaxID>0123456789</TaxID></Seller>
</Invoice>`)

	result := p.ProcessXMLBytes(ctx, xmlData)
	require.Nil(t, result.Error)
	require.NotNil(t, result.Invoice)
	assert.Equal(t, "0000002", result.Invoice.Number)
}

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected processor.Format
	}{
		{
			name:     "XML with declaration",
			data:     []byte(`<?xml version="1.0"?><Invoice/>`),
			expected: processor.FormatXML,
		},
		{
			name:     "XML without declaration",
			data:     []byte(`<Invoice><Number>1</Number></Invoice>`),
			expected: processor.FormatXML,
		},
		{
			name:     "PDF",
			data:     []byte("%PDF-1.4\n%some content"),
			expected: processor.FormatPDF,
		},
		{
			name:     "PNG image",
			data:     []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A},
			expected: processor.FormatImage,
		},
		{
			name:     "JPEG image",
			data:     []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46},
			expected: processor.FormatImage,
		},
		{
			name:     "TIFF little-endian",
			data:     []byte{0x49, 0x49, 0x2A, 0x00, 0x08, 0x00, 0x00, 0x00},
			expected: processor.FormatImage,
		},
		{
			name:     "TIFF big-endian",
			data:     []byte{0x4D, 0x4D, 0x00, 0x2A, 0x00, 0x00, 0x00, 0x08},
			expected: processor.FormatImage,
		},
		{
			name:     "Unknown format",
			data:     []byte("some random text"),
			expected: processor.FormatUnknown,
		},
		{
			name:     "Empty data",
			data:     []byte{},
			expected: processor.FormatUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format := processor.DetectFormat(tt.data)
			assert.Equal(t, tt.expected, format)
		})
	}
}

func TestFormatString(t *testing.T) {
	tests := []struct {
		format   processor.Format
		expected string
	}{
		{processor.FormatXML, "xml"},
		{processor.FormatPDF, "pdf"},
		{processor.FormatImage, "image"},
		{processor.FormatUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.format.String())
		})
	}
}

func TestExtractionMethod(t *testing.T) {
	assert.Equal(t, processor.ExtractionMethod("xml"), processor.MethodXML)
	assert.Equal(t, processor.ExtractionMethod("llm_text"), processor.MethodLLMText)
	assert.Equal(t, processor.ExtractionMethod("llm_vision"), processor.MethodLLMVision)
}

func TestProcessImage_NoLLM(t *testing.T) {
	ctx := context.Background()
	p := processor.NewPipeline() // No LLM extractor

	result := p.ProcessImage(ctx, []byte("fake image"), "image/png")
	require.NotNil(t, result.Error)
	assert.Contains(t, result.Error.Error(), "LLM extractor not configured")
}

func TestResult_Fields(t *testing.T) {
	result := &processor.Result{
		Invoice:    nil,
		Method:     processor.MethodLLMText,
		Confidence: 0.85,
		Warnings:   []string{"warning1", "warning2"},
		Error:      nil,
	}

	assert.Equal(t, processor.MethodLLMText, result.Method)
	assert.Equal(t, 0.85, result.Confidence)
	assert.Len(t, result.Warnings, 2)
}

// Benchmark tests

func BenchmarkDetectFormat_XML(b *testing.B) {
	data := []byte(`<?xml version="1.0"?><Invoice><Number>1</Number></Invoice>`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processor.DetectFormat(data)
	}
}

func BenchmarkDetectFormat_PDF(b *testing.B) {
	data := []byte("%PDF-1.4\n%some content here")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processor.DetectFormat(data)
	}
}

func BenchmarkProcessXML(b *testing.B) {
	ctx := context.Background()
	p := processor.NewPipeline()
	xmlData := `<?xml version="1.0"?><Invoice><InvoiceNumber>001</InvoiceNumber><TaxID>0123456789</TaxID></Invoice>`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.ProcessXML(ctx, strings.NewReader(xmlData))
	}
}
