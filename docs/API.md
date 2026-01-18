# Invoice Processor API Reference

## Overview

The Invoice Processor provides a REST API for extracting structured data from invoices and POS receipts. It supports multiple input formats and automatically detects document types.

**Base URL:** `http://localhost:8080/api/v1`

## Authentication

Configure LLM API key via environment variable or server config for PDF/image processing.

## Endpoints

### Health Check

```
GET /health
```

Returns server health status.

**Response:**
```json
{
  "status": "ok",
  "time": "2025-01-18T12:00:00Z"
}
```

---

### Process Auto (Recommended)

```
POST /api/v1/process/auto
```

Auto-detects format and document type (invoice vs receipt). This is the recommended endpoint for most use cases.

**Headers:**
- `Content-Type`: `application/xml`, `application/pdf`, `image/png`, `image/jpeg`, `image/tiff`

**Request Body:** Raw file content (binary)

**Response:** [ProcessResponse](#processresponse)

**Example:**
```bash
curl -X POST http://localhost:8080/api/v1/process/auto \
  -H "Content-Type: image/jpeg" \
  --data-binary @receipt.jpg
```

---

### Process XML

```
POST /api/v1/process/xml
```

Process Vietnamese e-invoice XML (TCT, VNPT, MISA, Viettel, FPT formats).

**Headers:**
- `Content-Type`: `application/xml`

**Request Body:** Raw XML content

**Response:** [ProcessResponse](#processresponse)

---

### Process PDF

```
POST /api/v1/process/pdf
```

Process PDF invoice/receipt using LLM extraction.

**Headers:**
- `Content-Type`: `application/pdf`

**Request Body:** Raw PDF content (binary)

**Response:** [ProcessResponse](#processresponse)

**Note:** Requires LLM API key configured.

---

### Process Image

```
POST /api/v1/process/image
```

Process invoice/receipt image using LLM vision.

**Headers:**
- `Content-Type`: `image/png`, `image/jpeg`, or `image/tiff`

**Request Body:** Raw image content (binary)

**Response:** [ProcessResponse](#processresponse)

**Note:** Requires LLM API key configured.

---

### Validate

```
POST /api/v1/validate
```

Validate XML invoice structure and data.

**Request Body:** Raw XML content

**Response:** [ValidationResponse](#validationresponse)

---

### Info

```
POST /api/v1/info
```

Get file format information without processing.

**Request Body:** Raw file content (binary)

**Response:** [InfoResponse](#inforesponse)

---

## Response Types

### ProcessResponse

Successful extraction response.

| Field | Type | Description |
|-------|------|-------------|
| `invoice` | [Document](#document-structure) | Extracted document data |
| `method` | string | Extraction method: `xml`, `llm_text`, `llm_vision` |
| `confidence` | float | Confidence score (0.0-1.0) |
| `warnings` | string[] | Optional extraction warnings |

**Confidence Scores:**
- `1.0` - XML parsing (deterministic)
- `0.85` - LLM text extraction
- `0.80` - LLM vision (invoice)
- `0.75` - LLM vision (receipt - lower due to thermal paper)

### ValidationResponse

| Field | Type | Description |
|-------|------|-------------|
| `valid` | boolean | Whether document is valid |
| `errors` | string[] | Validation errors |
| `warnings` | string[] | Validation warnings |

### InfoResponse

| Field | Type | Description |
|-------|------|-------------|
| `format` | string | `xml`, `pdf`, `image`, or `unknown` |
| `mime_type` | string | Detected MIME type |
| `size` | int | File size in bytes |

### ErrorResponse

| Field | Type | Description |
|-------|------|-------------|
| `error` | string | Error message |
| `details` | string | Optional error details |
| `warnings` | string[] | Optional warnings |

---

## Document Structure

The unified `Document` object represents both invoices and receipts.

### Root Object

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | No | Unique identifier |
| `document_type` | string | Yes | `"invoice"` or `"receipt"` |
| `number` | string | Yes | Document number |
| `series` | string | No | Invoice series (2-5 chars) |
| `date` | datetime | Yes | Document date (ISO 8601) |
| `type` | string | No | `"Normal"`, `"Replacement"`, `"Adjustment"` |
| `provider` | string | No | E-invoice provider (TCT, VNPT, etc.) |
| `seller` | [Party](#party) | Yes | Seller/store information |
| `buyer` | [Party](#party) | No | Buyer information |
| `items` | [LineItem[]](#lineitem) | Yes | Line items |
| `subtotal_amount` | decimal | Yes | Pre-tax total |
| `tax_amount` | decimal | No | Total tax/VAT |
| `total_amount` | decimal | Yes | Final total |
| `currency` | string | Yes | Currency code (default: `"VND"`) |
| `exchange_rate` | decimal | No | For foreign currency |
| `remarks` | string | No | Notes/remarks |
| `payment_terms` | string | No | Payment terms |

### Receipt-Specific Fields

| Field | Type | Description |
|-------|------|-------------|
| `cashier` | string | Cashier name |
| `terminal_id` | string | POS terminal ID |
| `payment_method` | string | `"CASH"`, `"CARD"`, `"E_WALLET"`, `"BANK_TRANSFER"` |
| `receipt_number` | string | Receipt/transaction number |
| `receipt_time` | string | Time in HH:MM format |
| `amount_tendered` | decimal | Cash given by customer |
| `change` | decimal | Change returned |

### Party

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Party name |
| `tax_id` | string | Invoice only | Tax ID (10 digits) |
| `address` | string | No | Address |
| `phone` | string | No | Phone number |
| `email` | string | No | Email address |
| `bank_account` | string | No | Bank account number |
| `bank_name` | string | No | Bank name |

### LineItem

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `number` | int | Yes | Line number |
| `code` | string | No | Item/SKU code |
| `name` | string | Yes | Item name |
| `description` | string | No | Description |
| `unit` | string | Yes | Unit of measure |
| `quantity` | decimal | Yes | Quantity |
| `unit_price` | decimal | Yes | Price per unit |
| `discount` | decimal | No | Discount percentage |
| `discount_amt` | decimal | No | Discount amount |
| `vat_rate` | int | No | VAT rate (0, 5, 10) |
| `amount` | decimal | Yes | Quantity × Unit Price |
| `vat_amount` | decimal | No | VAT amount |
| `total` | decimal | Yes | Line total |

---

## Examples

### Process Invoice Image

```bash
curl -X POST http://localhost:8080/api/v1/process/auto \
  -H "Content-Type: image/png" \
  --data-binary @invoice.png
```

**Response:**
```json
{
  "invoice": {
    "document_type": "invoice",
    "number": "0001234",
    "series": "AB",
    "date": "2025-01-15T00:00:00Z",
    "seller": {
      "name": "CÔNG TY TNHH ABC",
      "tax_id": "0123456789",
      "address": "123 Nguyễn Văn A, Q1, TP.HCM"
    },
    "buyer": {
      "name": "Nguyễn Văn B",
      "tax_id": "9876543210"
    },
    "items": [
      {
        "number": 1,
        "name": "Sản phẩm A",
        "unit": "cái",
        "quantity": "2",
        "unit_price": "100000",
        "vat_rate": 10,
        "amount": "200000",
        "vat_amount": "20000",
        "total": "220000"
      }
    ],
    "subtotal_amount": "200000",
    "tax_amount": "20000",
    "total_amount": "220000",
    "currency": "VND"
  },
  "method": "llm_vision",
  "confidence": 0.80
}
```

### Process POS Receipt

```bash
curl -X POST http://localhost:8080/api/v1/process/auto \
  -H "Content-Type: image/jpeg" \
  --data-binary @receipt.jpg
```

**Response:**
```json
{
  "invoice": {
    "document_type": "receipt",
    "number": "0012345",
    "date": "2025-01-18T00:00:00Z",
    "seller": {
      "name": "AMERICANO COFFEE",
      "address": "123 Lê Lợi, Q1, TP.HCM"
    },
    "items": [
      {
        "number": 1,
        "name": "CAFE PHA KIỂU MỸ ĐÁ",
        "unit": "ly",
        "quantity": "1",
        "unit_price": "59000",
        "amount": "59000",
        "total": "59000"
      }
    ],
    "subtotal_amount": "59000",
    "total_amount": "59000",
    "currency": "VND",
    "payment_method": "CASH",
    "cashier": "Thu Hà",
    "receipt_time": "14:32",
    "amount_tendered": "100000",
    "change": "41000"
  },
  "method": "llm_vision",
  "confidence": 0.75
}
```

---

## Error Codes

| HTTP Status | Description |
|-------------|-------------|
| 200 | Success |
| 400 | Bad Request - Invalid input |
| 422 | Unprocessable Entity - Extraction failed |
| 500 | Internal Server Error |

---

## CLI Usage

The same functionality is available via CLI:

```bash
# Process single file
invoice-processor process receipt.jpg

# Process with verbose output
invoice-processor process -v invoice.pdf

# Process multiple files
invoice-processor process *.xml -f table

# Output to JSON file
invoice-processor process invoices/ -o results.json
```

See `invoice-processor --help` for full CLI documentation.
