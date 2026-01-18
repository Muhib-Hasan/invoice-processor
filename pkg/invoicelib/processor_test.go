package invoicelib_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rezonia/invoice-processor/pkg/invoicelib"
)

func TestNewProcessor(t *testing.T) {
	opts := invoicelib.DefaultPipelineOptions()
	opts.EnableLLM = false

	proc := invoicelib.NewProcessor(opts)
	require.NotNil(t, proc)
}

func TestNewDefaultProcessor(t *testing.T) {
	proc := invoicelib.NewDefaultProcessor()
	require.NotNil(t, proc)
}

func TestDefaultPipelineOptions(t *testing.T) {
	opts := invoicelib.DefaultPipelineOptions()

	assert.Equal(t, 0.90, opts.TemplateThreshold)
	assert.Equal(t, 0.85, opts.LLMThreshold)
	assert.Equal(t, 0.70, opts.ReviewThreshold)
	assert.True(t, opts.EnableLLM)
	assert.True(t, opts.EnableOCR)
	assert.True(t, opts.ValidateAfterExtraction)
	assert.Equal(t, "https://openrouter.ai/api/v1", opts.LLMBaseURL)
	assert.Equal(t, "anthropic/claude-3.5-sonnet", opts.LLMModel)
	assert.Equal(t, "anthropic/claude-3.5-sonnet", opts.LLMVisionModel)
}

func TestProcessorProcessXML(t *testing.T) {
	opts := invoicelib.DefaultPipelineOptions()
	opts.EnableLLM = false
	proc := invoicelib.NewProcessor(opts)

	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<Invoice>
	<InvoiceNo>0000001</InvoiceNo>
	<InvoiceSeries>KK23</InvoiceSeries>
	<IssueDate>2024-01-15</IssueDate>
	<Seller>
		<TaxID>0123456789</TaxID>
		<Name>Test Seller Company</Name>
		<Address>123 Seller Street</Address>
	</Seller>
	<Buyer>
		<TaxID>9876543210</TaxID>
		<Name>Test Buyer Company</Name>
	</Buyer>
	<TotalAmount>1000000</TotalAmount>
	<Currency>VND</Currency>
</Invoice>`

	result, err := proc.ProcessXML(context.Background(), bytes.NewReader([]byte(xmlData)))
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "0000001", result.Invoice.Number)
	assert.Equal(t, "KK23", result.Invoice.Series)
	assert.Equal(t, "xml", result.Method)
	assert.Equal(t, 1.0, result.Confidence)
}

func TestProcessorProcess_AutoDetectXML(t *testing.T) {
	opts := invoicelib.DefaultPipelineOptions()
	opts.EnableLLM = false
	proc := invoicelib.NewProcessor(opts)

	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<Invoice>
	<InvoiceNo>0000002</InvoiceNo>
	<Seller><TaxID>1234567890</TaxID></Seller>
	<TotalAmount>500000</TotalAmount>
</Invoice>`

	result, err := proc.Process(context.Background(), bytes.NewReader([]byte(xmlData)))
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "0000002", result.Invoice.Number)
	assert.Equal(t, "xml", result.Method)
}

func TestProcessorProcess_InvalidFormat(t *testing.T) {
	opts := invoicelib.DefaultPipelineOptions()
	opts.EnableLLM = false
	proc := invoicelib.NewProcessor(opts)

	// Random binary data that's not a known format
	data := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05}

	_, err := proc.Process(context.Background(), bytes.NewReader(data))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported")
}

func TestProcessorProcessXML_InvalidXML(t *testing.T) {
	opts := invoicelib.DefaultPipelineOptions()
	opts.EnableLLM = false
	proc := invoicelib.NewProcessor(opts)

	_, err := proc.ProcessXML(context.Background(), bytes.NewReader([]byte("not xml")))
	require.Error(t, err)
}

func TestProcessorProcessBatch(t *testing.T) {
	opts := invoicelib.DefaultPipelineOptions()
	opts.EnableLLM = false
	proc := invoicelib.NewProcessor(opts)

	xml1 := `<?xml version="1.0"?><Invoice><InvoiceNo>0001</InvoiceNo><Seller><TaxID>1111111111</TaxID></Seller></Invoice>`
	xml2 := `<?xml version="1.0"?><Invoice><InvoiceNo>0002</InvoiceNo><Seller><TaxID>2222222222</TaxID></Seller></Invoice>`

	inputs := []interface{}{
		bytes.NewReader([]byte(xml1)),
		bytes.NewReader([]byte(xml2)),
	}

	// Convert to io.Reader slice
	readers := make([]interface{}, len(inputs))
	copy(readers, inputs)

	// Process first file to verify it works
	result, err := proc.Process(context.Background(), bytes.NewReader([]byte(xml1)))
	require.NoError(t, err)
	assert.Equal(t, "0001", result.Invoice.Number)

	result, err = proc.Process(context.Background(), bytes.NewReader([]byte(xml2)))
	require.NoError(t, err)
	assert.Equal(t, "0002", result.Invoice.Number)
}

func TestExtractionResult_NeedsReview(t *testing.T) {
	opts := invoicelib.DefaultPipelineOptions()
	opts.EnableLLM = false
	opts.ReviewThreshold = 0.90 // Set high threshold to trigger review flag
	proc := invoicelib.NewProcessor(opts)

	xmlData := `<?xml version="1.0"?><Invoice><InvoiceNo>001</InvoiceNo><Seller><TaxID>1234567890</TaxID></Seller></Invoice>`

	result, err := proc.ProcessXML(context.Background(), bytes.NewReader([]byte(xmlData)))
	require.NoError(t, err)

	// XML parsing has 1.0 confidence, so NeedsReview should be false with 0.90 threshold
	assert.False(t, result.NeedsReview)
}

// Test re-exported types
func TestReExportedTypes(t *testing.T) {
	// Verify that types are properly re-exported
	var invoice invoicelib.Invoice
	invoice.Number = "12345"
	assert.Equal(t, "12345", invoice.Number)

	var party invoicelib.Party
	party.TaxID = "0123456789"
	assert.Equal(t, "0123456789", party.TaxID)

	var item invoicelib.LineItem
	item.Name = "Test Item"
	assert.Equal(t, "Test Item", item.Name)

	// Test provider constants
	assert.Equal(t, invoicelib.ProviderTCT, invoicelib.Provider("TCT"))
	assert.Equal(t, invoicelib.ProviderVNPT, invoicelib.Provider("VNPT"))
	assert.Equal(t, invoicelib.ProviderMISA, invoicelib.Provider("MISA"))
	assert.Equal(t, invoicelib.ProviderViettel, invoicelib.Provider("VIETTEL"))
	assert.Equal(t, invoicelib.ProviderFPT, invoicelib.Provider("FPT"))

	// Test VAT rates
	assert.Equal(t, invoicelib.VATRate(0), invoicelib.VATRate0)
	assert.Equal(t, invoicelib.VATRate(5), invoicelib.VATRate5)
	assert.Equal(t, invoicelib.VATRate(10), invoicelib.VATRate10)

	// Test invoice types
	assert.Equal(t, invoicelib.InvoiceType("Normal"), invoicelib.InvoiceTypeNormal)
	assert.Equal(t, invoicelib.InvoiceType("Replacement"), invoicelib.InvoiceTypeReplacement)
	assert.Equal(t, invoicelib.InvoiceType("Adjustment"), invoicelib.InvoiceTypeAdjustment)
}
