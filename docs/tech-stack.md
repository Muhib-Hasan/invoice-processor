# Tech Stack: Go Vietnam Invoice Processor

**Date**: 2026-01-18 | **Status**: Recommended

---

## 1. Core Dependencies

| Category | Package | Rationale |
|----------|---------|-----------|
| **PDF Parsing** | `pdfcpu/pdfcpu` | Pure Go, text extraction w/ positions, zone-based parsing |
| **XML Parsing** | `encoding/xml` | Stdlib sufficient, custom struct tags for provider variants |
| **Decimals** | `shopspring/decimal` | Financial precision, avoid float64 rounding |
| **CLI** | `spf13/cobra` | De facto standard, subcommands, flags |
| **Logging** | `log/slog` | Stdlib (Go 1.21+), structured, zero deps |
| **Testing** | `stretchr/testify` | Assert/require/mock, 636k dependents |

---

## 2. LLM Integration (OpenRouter Universal API)

**Pattern**: Single unified API via OpenRouter

```go
// OpenRouter provides access to Claude, GPT-4, Gemini, Llama via single endpoint
type LLMClient struct {
    apiKey string
    model  string // e.g., "anthropic/claude-3.5-sonnet", "openai/gpt-4o"
}
```

| Aspect | Details |
|--------|---------|
| **Gateway** | OpenRouter API (`https://openrouter.ai/api/v1`) |
| **Go Client** | Standard `net/http` (OpenAI-compatible endpoint) |
| **Models** | Claude 3.5 Sonnet, GPT-4o, Gemini Pro, Llama 3.1, etc. |
| **Benefits** | Single API key, automatic fallbacks, cost tracking |

**Rationale**: OpenRouter unifies 100+ models under one API. Simplifies codebase - no need for multiple SDKs. Model switching via config, not code changes.

---

## 3. OCR Solution

**Strategy**: Hybrid with fallback chain

| Priority | Method | When to Use | Cost |
|----------|--------|-------------|------|
| 1 | Template matching (pdfcpu) | Known formats | $0 |
| 2 | Claude Vision | Unknown/complex | $0.01/page |
| 3 | Tesseract (`otiai10/gosseract`) | Scanned legacy | Self-hosted |

**No cloud OCR initially** (Textract/Vision). Claude Vision handles unstructured better at lower cost.

---

## 4. Project Structure

```
invoice-processor/
├── cmd/invoice-cli/      # CLI entry point (uses pkg/*)
├── pkg/invoicelib/       # Public API (Parser, Validator interfaces)
├── internal/
│   ├── parser/           # PDF/XML parsing
│   ├── extractor/        # Template matching, LLM calls
│   ├── provider/         # VNPT, MISA, Viettel adapters
│   └── errors/           # Custom error types
├── test/fixtures/        # Sample invoices (.pdf, .xml)
└── go.mod
```

---

## 5. Additional Dependencies

| Package | Purpose |
|---------|---------|
| `google/uuid` | Invoice ID generation |
| `go-playground/validator` | Struct validation (optional, can use custom) |

---

## 6. go.mod Dependencies (Complete)

```go
require (
    // Core
    github.com/pdfcpu/pdfcpu v0.8.0
    github.com/shopspring/decimal v1.4.0

    // CLI
    github.com/spf13/cobra v1.8.1

    // API
    github.com/go-chi/chi/v5 v5.0.12        // REST router
    google.golang.org/grpc v1.62.0          // gRPC
    google.golang.org/protobuf v1.33.0      // Protocol Buffers

    // Testing
    github.com/stretchr/testify v1.9.0

    // Optional
    github.com/otiai10/gosseract/v2 v2.4.1  // OCR for scanned PDFs
)
// Note: OpenRouter uses OpenAI-compatible API - standard net/http is sufficient
```

---

## 7. Key Decisions

1. **OpenRouter for LLM** - Single API for all providers (Claude, GPT-4, Gemini), no multiple SDKs
2. **No external OCR cloud services** - LLM Vision sufficient, cheaper
3. **Stdlib XML** - No need for etree/xmlpath complexity
4. **Hybrid PDF strategy** - Templates first, LLM fallback
5. **Minimal deps** - 5 direct dependencies total (reduced from 8)

---

## Unresolved

- Tesseract optional (only if scanned PDF volume justifies)
- `go-playground/validator` vs custom validation (defer until needed)
