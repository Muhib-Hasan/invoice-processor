package llm_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rezonia/invoice-processor/internal/llm"
)

func TestNewClient(t *testing.T) {
	client := llm.NewClient("test-api-key")
	require.NotNil(t, client)
}

func TestNewClient_WithOptions(t *testing.T) {
	client := llm.NewClient("test-api-key",
		llm.WithBaseURL("https://custom.api.com/v1"),
		llm.WithDefaultModel(llm.ModelGPT4o),
	)
	require.NotNil(t, client)
}

func TestNewExtractor(t *testing.T) {
	client := llm.NewClient("test-api-key")
	extractor := llm.NewExtractor(client)
	require.NotNil(t, extractor)
}

func TestNewExtractor_WithModel(t *testing.T) {
	client := llm.NewClient("test-api-key")
	extractor := llm.NewExtractor(client, llm.WithModel(llm.ModelGPT4oMini))
	require.NotNil(t, extractor)
}

func TestExtractJSON_CodeBlock(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "json code block",
			input: "Here is the invoice data:\n```json\n{\"invoice_number\": \"001\"}\n```",
			expected: `{"invoice_number": "001"}`,
		},
		{
			name: "generic code block",
			input: "```\n{\"invoice_number\": \"002\"}\n```",
			expected: `{"invoice_number": "002"}`,
		},
		{
			name: "raw json object",
			input: `{"invoice_number": "003"}`,
			expected: `{"invoice_number": "003"}`,
		},
		{
			name: "raw json array",
			input: `[{"id": 1}, {"id": 2}]`,
			expected: `[{"id": 1}, {"id": 2}]`,
		},
		{
			name: "json with explanation",
			input: "I found the following data:\n```json\n{\"total\": 1000000}\n```\nThis represents the total amount.",
			expected: `{"total": 1000000}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := llm.ExtractJSON(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestModelConstants(t *testing.T) {
	models := []string{
		llm.ModelClaude35Sonnet,
		llm.ModelClaude3Haiku,
		llm.ModelGPT4oMini,
		llm.ModelGPT4o,
		llm.ModelGeminiFlash,
	}

	for _, m := range models {
		assert.NotEmpty(t, m)
		assert.Contains(t, m, "/") // All models have provider/model format
	}
}

// Note: ChatRequest and Message marshaling tests removed after SDK migration.
// The openai-go SDK handles message construction internally.

func TestLLMResponse_Parsing(t *testing.T) {
	jsonResp := `{
		"invoice_number": "0000001",
		"series": "KK23",
		"date": "2026-01-18",
		"type": "normal",
		"seller": {
			"name": "ABC Company",
			"tax_id": "0123456789",
			"address": "123 Main St"
		},
		"buyer": {
			"name": "XYZ Corp",
			"tax_id": "9876543210"
		},
		"items": [
			{
				"number": 1,
				"name": "Product A",
				"quantity": 10,
				"unit_price": 100000,
				"amount": 1000000,
				"vat_rate": 10,
				"vat_amount": 100000,
				"total": 1100000
			}
		],
		"subtotal": 1000000,
		"total_vat": 100000,
		"total_amount": 1100000,
		"currency": "VND"
	}`

	var resp llm.LLMResponse
	err := json.Unmarshal([]byte(jsonResp), &resp)
	require.NoError(t, err)

	assert.Equal(t, "0000001", resp.InvoiceNumber)
	assert.Equal(t, "KK23", resp.Series)
	assert.Equal(t, "2026-01-18", resp.Date)
	assert.Equal(t, "ABC Company", resp.Seller.Name)
	assert.Equal(t, "0123456789", resp.Seller.TaxID)
	assert.Len(t, resp.Items, 1)
	assert.Equal(t, "Product A", resp.Items[0].Name)
}

func TestPromptTemplates(t *testing.T) {
	// Verify prompt templates are not empty
	assert.NotEmpty(t, llm.SystemPromptInvoiceExtractor)
	assert.NotEmpty(t, llm.UserPromptTextExtraction)
	assert.NotEmpty(t, llm.UserPromptImageExtraction)
	assert.NotEmpty(t, llm.UserPromptOCRCorrection)

	// Verify templates contain expected content
	assert.Contains(t, llm.SystemPromptInvoiceExtractor, "Vietnamese")
	assert.Contains(t, llm.SystemPromptInvoiceExtractor, "invoice")
	assert.Contains(t, llm.UserPromptTextExtraction, "JSON")
	assert.Contains(t, llm.UserPromptImageExtraction, "JSON")
}

func TestDefaultBaseURL(t *testing.T) {
	assert.Equal(t, "https://openrouter.ai/api/v1", llm.DefaultBaseURL)
}

// Benchmark tests

func BenchmarkExtractJSON(b *testing.B) {
	input := "Here is the data:\n```json\n{\"invoice_number\": \"001\", \"total\": 1000000}\n```"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		llm.ExtractJSON(input)
	}
}

// Note: BenchmarkChatRequestMarshal removed after SDK migration.
// The openai-go SDK handles request marshaling internally.
