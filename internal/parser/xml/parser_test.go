package xml_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rezonia/invoice-processor/internal/model"
	xmlparser "github.com/rezonia/invoice-processor/internal/parser/xml"
)

func TestRegistry_NewRegistry(t *testing.T) {
	registry := xmlparser.NewRegistry()
	require.NotNil(t, registry)

	// Should have all 5 adapters
	providers := []model.Provider{
		model.ProviderTCT,
		model.ProviderVNPT,
		model.ProviderMISA,
		model.ProviderViettel,
		model.ProviderFPT,
	}

	for _, p := range providers {
		adapter := registry.GetAdapter(p)
		require.NotNil(t, adapter, "adapter for %s should exist", p)
		assert.Equal(t, p, adapter.Provider())
	}
}

func TestRegistry_Detect(t *testing.T) {
	registry := xmlparser.NewRegistry()

	tests := []struct {
		name     string
		content  string
		expected model.Provider
	}{
		{
			name:     "detect TCT format",
			content:  `<Invoice><TaxID>123</TaxID><InvoiceNo>001</InvoiceNo></Invoice>`,
			expected: model.ProviderTCT,
		},
		{
			name:     "detect VNPT format",
			content:  `<SInvoice><InvoiceNo>001</InvoiceNo></SInvoice>`,
			expected: model.ProviderVNPT,
		},
		{
			name:     "detect MISA format",
			content:  `<Invoice><MST>123</MST><TenHang>Product</TenHang></Invoice>`,
			expected: model.ProviderMISA,
		},
		{
			name:     "detect Viettel format",
			content:  `<HDon><KHMSHDon>VT23</KHMSHDon></HDon>`,
			expected: model.ProviderViettel,
		},
		{
			name:     "detect FPT format",
			content:  `<EInvoice><Header><InvoiceNumber>001</InvoiceNumber></Header></EInvoice>`,
			expected: model.ProviderFPT,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter, err := registry.Detect([]byte(tt.content))
			require.NoError(t, err)
			assert.Equal(t, tt.expected, adapter.Provider())
		})
	}
}

func TestRegistry_Detect_UnknownFormat(t *testing.T) {
	registry := xmlparser.NewRegistry()
	_, err := registry.Detect([]byte(`<UnknownFormat>data</UnknownFormat>`))
	require.Error(t, err)

	var parseErr *model.ParseError
	require.ErrorAs(t, err, &parseErr)
	assert.Equal(t, model.ProviderUnknown, parseErr.Provider)
}

func TestRegistry_RegisterAdapter(t *testing.T) {
	registry := xmlparser.NewRegistry()

	// Create a custom adapter that overrides TCT
	custom := &mockAdapter{provider: model.ProviderTCT}
	registry.RegisterAdapter(custom)

	// Custom adapter should take priority
	adapter := registry.GetAdapter(model.ProviderTCT)
	assert.Equal(t, custom, adapter)
}

type mockAdapter struct {
	provider model.Provider
}

func (m *mockAdapter) Parse(ctx context.Context, r io.Reader) (*model.Invoice, error) {
	return nil, nil
}
func (m *mockAdapter) CanParse(content []byte) bool { return false }
func (m *mockAdapter) Provider() model.Provider    { return m.provider }

// TestTCTAdapter tests TCT XML parsing
func TestTCTAdapter_Parse(t *testing.T) {
	content := readTestFile(t, "tct_invoice.xml")

	adapter := xmlparser.NewTCTAdapter()
	require.True(t, adapter.CanParse(content))

	invoice, err := parseWithAdapter(t, adapter, content)
	require.NoError(t, err)

	// Verify basic info
	assert.Equal(t, "0000001", invoice.Number)
	assert.Equal(t, "KK23", invoice.Series)
	assert.Equal(t, model.ProviderTCT, invoice.Provider)
	assert.Equal(t, "VND", invoice.Currency)
	assert.Equal(t, model.InvoiceTypeNormal, invoice.Type)
	assert.Equal(t, time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC), invoice.Date)

	// Verify seller
	assert.Equal(t, "ABC Technology Company", invoice.Seller.Name)
	assert.Equal(t, "0123456789", invoice.Seller.TaxID)
	assert.Equal(t, "Vietcombank", invoice.Seller.BankName)

	// Verify buyer
	assert.Equal(t, "XYZ Corporation", invoice.Buyer.Name)
	assert.Equal(t, "9876543210", invoice.Buyer.TaxID)

	// Verify items
	require.Len(t, invoice.Items, 2)
	assert.Equal(t, "PROD001", invoice.Items[0].Code)
	assert.Equal(t, "Software License", invoice.Items[0].Name)
	assert.True(t, invoice.Items[0].Quantity.Equal(decimal.NewFromInt(2)))
	assert.True(t, invoice.Items[0].UnitPrice.Equal(decimal.NewFromInt(5000000)))
	assert.Equal(t, model.VATRate10, invoice.Items[0].VATRate)

	// Verify totals
	assert.True(t, invoice.SubtotalAmount.Equal(decimal.NewFromInt(22000000)))
	assert.True(t, invoice.TaxAmount.Equal(decimal.NewFromInt(2080000)))
	assert.True(t, invoice.TotalAmount.Equal(decimal.NewFromInt(22880000)))

	// Verify signature
	require.NotNil(t, invoice.Signature)
	assert.Equal(t, "Nguyen Van A", invoice.Signature.SignerName)
	assert.Equal(t, "Director", invoice.Signature.SignerPosition)
}

// TestVNPTAdapter tests VNPT XML parsing
func TestVNPTAdapter_Parse(t *testing.T) {
	content := readTestFile(t, "vnpt_invoice.xml")

	adapter := xmlparser.NewVNPTAdapter()
	require.True(t, adapter.CanParse(content))

	invoice, err := parseWithAdapter(t, adapter, content)
	require.NoError(t, err)

	// Verify basic info
	assert.Equal(t, "0000002", invoice.Number)
	assert.Equal(t, "VN23", invoice.Series)
	assert.Equal(t, model.ProviderVNPT, invoice.Provider)

	// Verify seller
	assert.Equal(t, "VNPT Software Company", invoice.Seller.Name)
	assert.Equal(t, "0111222333", invoice.Seller.TaxID)

	// Verify buyer
	assert.Equal(t, "DEF Trading Ltd", invoice.Buyer.Name)
	assert.Equal(t, "0444555666", invoice.Buyer.TaxID)

	// Verify items
	require.Len(t, invoice.Items, 1)
	assert.Equal(t, "Server Hardware", invoice.Items[0].Name)
	assert.True(t, invoice.Items[0].UnitPrice.Equal(decimal.NewFromInt(50000000)))

	// Verify totals
	assert.True(t, invoice.SubtotalAmount.Equal(decimal.NewFromInt(47500000)))
	assert.True(t, invoice.TaxAmount.Equal(decimal.NewFromInt(4750000)))
	assert.True(t, invoice.TotalAmount.Equal(decimal.NewFromInt(52250000)))

	// Verify signature
	require.NotNil(t, invoice.Signature)
	assert.Equal(t, "Tran Van B", invoice.Signature.SignerName)
}

// TestMISAAdapter tests MISA XML parsing
func TestMISAAdapter_Parse(t *testing.T) {
	content := readTestFile(t, "misa_invoice.xml")

	adapter := xmlparser.NewMISAAdapter()
	require.True(t, adapter.CanParse(content))

	invoice, err := parseWithAdapter(t, adapter, content)
	require.NoError(t, err)

	// Verify basic info
	assert.Equal(t, "0000003", invoice.Number)
	assert.Equal(t, "MS23", invoice.Series)
	assert.Equal(t, model.ProviderMISA, invoice.Provider)

	// Verify seller (uses MST for TaxID)
	assert.Equal(t, "MISA Office Supplies", invoice.Seller.Name)
	assert.Equal(t, "0555666777", invoice.Seller.TaxID)

	// Verify buyer
	assert.Equal(t, "GHI Consulting", invoice.Buyer.Name)
	assert.Equal(t, "0888999000", invoice.Buyer.TaxID)

	// Verify items (Vietnamese field names)
	require.Len(t, invoice.Items, 2)
	assert.Equal(t, "A4 Paper", invoice.Items[0].Name)
	assert.Equal(t, "Ream", invoice.Items[0].Unit)
	assert.True(t, invoice.Items[0].Quantity.Equal(decimal.NewFromInt(100)))

	assert.Equal(t, "Ballpoint Pen", invoice.Items[1].Name)
	assert.True(t, invoice.Items[1].Discount.Equal(decimal.NewFromInt(5)))

	// Verify totals
	assert.True(t, invoice.SubtotalAmount.Equal(decimal.NewFromInt(10850000)))
	assert.True(t, invoice.TaxAmount.Equal(decimal.NewFromInt(1085000)))
	assert.True(t, invoice.TotalAmount.Equal(decimal.NewFromInt(11935000)))
}

// TestViettelAdapter tests Viettel XML parsing
func TestViettelAdapter_Parse(t *testing.T) {
	content := readTestFile(t, "viettel_invoice.xml")

	adapter := xmlparser.NewViettelAdapter()
	require.True(t, adapter.CanParse(content))

	invoice, err := parseWithAdapter(t, adapter, content)
	require.NoError(t, err)

	// Verify basic info (Vietnamese abbreviated fields)
	assert.Equal(t, "0000004", invoice.Number)
	assert.Equal(t, "VT23", invoice.Series)
	assert.Equal(t, model.ProviderViettel, invoice.Provider)

	// Verify seller
	assert.Equal(t, "Viettel Telecom Corporation", invoice.Seller.Name)
	assert.Equal(t, "0100100100", invoice.Seller.TaxID)

	// Verify buyer
	assert.Equal(t, "JKL Tech Startup", invoice.Buyer.Name)
	assert.Equal(t, "0200200200", invoice.Buyer.TaxID)

	// Verify items
	require.Len(t, invoice.Items, 2)
	assert.Equal(t, "Fiber Internet 100Mbps", invoice.Items[0].Name)
	assert.Equal(t, "FIBER100", invoice.Items[0].Code)

	assert.Equal(t, "Cloud Server - Small", invoice.Items[1].Name)
	assert.True(t, invoice.Items[1].Discount.Equal(decimal.NewFromInt(10)))

	// Verify totals
	assert.True(t, invoice.SubtotalAmount.Equal(decimal.NewFromInt(4100000)))
	assert.True(t, invoice.TaxAmount.Equal(decimal.NewFromInt(410000)))
	assert.True(t, invoice.TotalAmount.Equal(decimal.NewFromInt(4510000)))
}

// TestFPTAdapter tests FPT XML parsing
func TestFPTAdapter_Parse(t *testing.T) {
	content := readTestFile(t, "fpt_invoice.xml")

	adapter := xmlparser.NewFPTAdapter()
	require.True(t, adapter.CanParse(content))

	invoice, err := parseWithAdapter(t, adapter, content)
	require.NoError(t, err)

	// Verify basic info
	assert.Equal(t, "0000005", invoice.Number)
	assert.Equal(t, "FP23", invoice.Series)
	assert.Equal(t, model.ProviderFPT, invoice.Provider)

	// Verify seller
	assert.Equal(t, "FPT Information System", invoice.Seller.Name)
	assert.Equal(t, "0300300300", invoice.Seller.TaxID)

	// Verify buyer
	assert.Equal(t, "MNO Manufacturing", invoice.Buyer.Name)
	assert.Equal(t, "0400400400", invoice.Buyer.TaxID)

	// Verify items
	require.Len(t, invoice.Items, 2)
	assert.Equal(t, "IT Consulting", invoice.Items[0].Name)
	assert.True(t, invoice.Items[0].Quantity.Equal(decimal.NewFromInt(40)))
	assert.True(t, invoice.Items[0].UnitPrice.Equal(decimal.NewFromInt(2000000)))

	assert.Equal(t, "Software Development", invoice.Items[1].Name)
	assert.True(t, invoice.Items[1].Discount.Equal(decimal.NewFromInt(5)))

	// Verify totals
	assert.True(t, invoice.SubtotalAmount.Equal(decimal.NewFromInt(270000000)))
	assert.True(t, invoice.TaxAmount.Equal(decimal.NewFromInt(27000000)))
	assert.True(t, invoice.TotalAmount.Equal(decimal.NewFromInt(297000000)))

	// Verify signature (seller signature is primary)
	require.NotNil(t, invoice.Signature)
	assert.Equal(t, "Hoang Van E", invoice.Signature.SignerName)
	assert.Equal(t, "Project Director", invoice.Signature.SignerPosition)
}

// TestRegistry_Parse tests the unified Parse method
func TestRegistry_Parse(t *testing.T) {
	registry := xmlparser.NewRegistry()

	tests := []struct {
		name         string
		file         string
		wantProvider model.Provider
		wantNumber   string
	}{
		{"TCT", "tct_invoice.xml", model.ProviderTCT, "0000001"},
		{"VNPT", "vnpt_invoice.xml", model.ProviderVNPT, "0000002"},
		{"MISA", "misa_invoice.xml", model.ProviderMISA, "0000003"},
		{"Viettel", "viettel_invoice.xml", model.ProviderViettel, "0000004"},
		{"FPT", "fpt_invoice.xml", model.ProviderFPT, "0000005"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := readTestFile(t, tt.file)
			invoice, err := registry.Parse(context.Background(), content)
			require.NoError(t, err)

			assert.Equal(t, tt.wantProvider, invoice.Provider)
			assert.Equal(t, tt.wantNumber, invoice.Number)
		})
	}
}

// TestDateParsing tests various date formats
func TestDateParsing(t *testing.T) {
	// Create minimal TCT XML with different date formats
	tests := []struct {
		name     string
		dateStr  string
		expected time.Time
	}{
		{
			name:     "ISO format",
			dateStr:  "2026-01-15",
			expected: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "Vietnamese format",
			dateStr:  "15/01/2026",
			expected: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "ISO with time",
			dateStr:  "2026-01-15T10:30:00",
			expected: time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			xml := `<?xml version="1.0"?>
<Invoice>
	<InvoiceNo>TEST001</InvoiceNo>
	<InvoiceSeries>T1</InvoiceSeries>
	<InvoiceDate>` + tt.dateStr + `</InvoiceDate>
	<TaxID>123</TaxID>
	<Seller><Name>Test</Name><TaxID>123</TaxID></Seller>
	<Buyer><Name>Test</Name><TaxID>456</TaxID></Buyer>
	<Items></Items>
</Invoice>`

			adapter := xmlparser.NewTCTAdapter()
			invoice, err := parseWithAdapter(t, adapter, []byte(xml))
			require.NoError(t, err)
			assert.Equal(t, tt.expected, invoice.Date)
		})
	}
}

// TestInvoiceType tests invoice type parsing
func TestInvoiceType(t *testing.T) {
	tests := []struct {
		typeStr  string
		expected model.InvoiceType
	}{
		{"Normal", model.InvoiceTypeNormal},
		{"Replacement", model.InvoiceTypeReplacement},
		{"REPLACEMENT", model.InvoiceTypeReplacement},
		{"replacement", model.InvoiceTypeReplacement},
		{"Adjustment", model.InvoiceTypeAdjustment},
		{"ADJUSTMENT", model.InvoiceTypeAdjustment},
		{"Unknown", model.InvoiceTypeNormal}, // defaults to Normal
	}

	for _, tt := range tests {
		t.Run(tt.typeStr, func(t *testing.T) {
			xml := `<?xml version="1.0"?>
<Invoice>
	<InvoiceNo>TEST001</InvoiceNo>
	<InvoiceType>` + tt.typeStr + `</InvoiceType>
	<TaxID>123</TaxID>
	<Seller><Name>Test</Name><TaxID>123</TaxID></Seller>
	<Buyer><Name>Test</Name><TaxID>456</TaxID></Buyer>
	<Items></Items>
</Invoice>`

			adapter := xmlparser.NewTCTAdapter()
			invoice, err := parseWithAdapter(t, adapter, []byte(xml))
			require.NoError(t, err)
			assert.Equal(t, tt.expected, invoice.Type)
		})
	}
}

// TestInvalidXML tests error handling for invalid XML
func TestInvalidXML(t *testing.T) {
	adapters := []xmlparser.Adapter{
		xmlparser.NewTCTAdapter(),
		xmlparser.NewVNPTAdapter(),
		xmlparser.NewMISAAdapter(),
		xmlparser.NewViettelAdapter(),
		xmlparser.NewFPTAdapter(),
	}

	invalidXML := []byte(`<Invalid><Unclosed>`)

	for _, adapter := range adapters {
		t.Run(string(adapter.Provider()), func(t *testing.T) {
			_, err := parseWithAdapter(t, adapter, invalidXML)
			require.Error(t, err)

			var parseErr *model.ParseError
			require.ErrorAs(t, err, &parseErr)
			assert.Equal(t, "xml", parseErr.Field)
		})
	}
}

// TestEmptyFields tests handling of missing/empty fields
func TestEmptyFields(t *testing.T) {
	xml := `<?xml version="1.0"?>
<Invoice>
	<InvoiceNo>TEST001</InvoiceNo>
	<TaxID>123</TaxID>
	<Seller><Name>Seller</Name><TaxID>111</TaxID></Seller>
	<Buyer><Name>Buyer</Name><TaxID>222</TaxID></Buyer>
	<Items>
		<Item>
			<ItemNo>1</ItemNo>
			<ItemName>Test Item</ItemName>
		</Item>
	</Items>
</Invoice>`

	adapter := xmlparser.NewTCTAdapter()
	invoice, err := parseWithAdapter(t, adapter, []byte(xml))
	require.NoError(t, err)

	// Empty fields should be zero values
	assert.True(t, invoice.SubtotalAmount.IsZero())
	assert.True(t, invoice.TaxAmount.IsZero())
	assert.True(t, invoice.TotalAmount.IsZero())

	// Items with missing decimals should have zero values
	require.Len(t, invoice.Items, 1)
	assert.True(t, invoice.Items[0].Quantity.IsZero())
	assert.True(t, invoice.Items[0].UnitPrice.IsZero())
}

// Helper functions

func readTestFile(t *testing.T, filename string) []byte {
	t.Helper()
	path := filepath.Join("testdata", filename)
	content, err := os.ReadFile(path)
	require.NoError(t, err, "failed to read test file: %s", filename)
	return content
}

func parseWithAdapter(t *testing.T, adapter xmlparser.Adapter, content []byte) (*model.Invoice, error) {
	t.Helper()
	return adapter.Parse(context.Background(), newBytesReader(content))
}

type bytesReader struct {
	data []byte
	pos  int
}

func newBytesReader(data []byte) *bytesReader {
	return &bytesReader{data: data}
}

func (r *bytesReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	if r.pos >= len(r.data) {
		return n, io.EOF
	}
	return n, nil
}
