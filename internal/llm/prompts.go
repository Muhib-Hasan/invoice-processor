package llm

// Invoice extraction prompts

const SystemPromptInvoiceExtractor = `You are an expert invoice data extractor specializing in Vietnamese e-invoices (hóa đơn điện tử).

Your task is to extract structured data from invoice text or images. The invoices may be in Vietnamese or English.

Common Vietnamese invoice terms:
- Hóa đơn = Invoice
- Số hóa đơn = Invoice number
- Ký hiệu = Series/Symbol
- Ngày = Date
- Mã số thuế (MST) = Tax ID
- Người bán/Bên bán = Seller
- Người mua/Bên mua = Buyer
- Địa chỉ = Address
- Tên hàng hóa/dịch vụ = Product/Service name
- Đơn vị tính = Unit
- Số lượng = Quantity
- Đơn giá = Unit price
- Thành tiền = Amount
- Thuế suất = Tax rate
- Tiền thuế = Tax amount
- Tổng cộng = Total
- Cộng tiền hàng = Subtotal
- Thuế GTGT = VAT

Extract ALL information you can find. If a field is not present, omit it from the output.
Always output valid JSON that matches the specified schema.
Numbers should be parsed as integers (for VND) or decimals.
Dates should be in ISO 8601 format (YYYY-MM-DD).`

const UserPromptTextExtraction = `Extract invoice data from the following text:

---
%s
---

Output JSON with this structure:
{
  "invoice_number": "string",
  "series": "string",
  "date": "YYYY-MM-DD",
  "type": "normal|replacement|adjustment",
  "seller": {
    "name": "string",
    "tax_id": "string",
    "address": "string",
    "phone": "string",
    "email": "string",
    "bank_account": "string",
    "bank_name": "string"
  },
  "buyer": {
    "name": "string",
    "tax_id": "string",
    "address": "string",
    "phone": "string",
    "email": "string"
  },
  "items": [
    {
      "number": 1,
      "code": "string",
      "name": "string",
      "unit": "string",
      "quantity": 1,
      "unit_price": 100000,
      "discount_percent": 0,
      "discount_amount": 0,
      "amount": 100000,
      "vat_rate": 10,
      "vat_amount": 10000,
      "total": 110000
    }
  ],
  "subtotal": 100000,
  "total_discount": 0,
  "total_vat": 10000,
  "total_amount": 110000,
  "currency": "VND",
  "payment_method": "string",
  "notes": "string"
}`

const UserPromptImageExtraction = `Extract invoice data from this invoice image.

Output JSON with this structure:
{
  "invoice_number": "string",
  "series": "string",
  "date": "YYYY-MM-DD",
  "type": "normal|replacement|adjustment",
  "seller": {
    "name": "string",
    "tax_id": "string",
    "address": "string",
    "phone": "string",
    "email": "string",
    "bank_account": "string",
    "bank_name": "string"
  },
  "buyer": {
    "name": "string",
    "tax_id": "string",
    "address": "string",
    "phone": "string",
    "email": "string"
  },
  "items": [
    {
      "number": 1,
      "code": "string",
      "name": "string",
      "unit": "string",
      "quantity": 1,
      "unit_price": 100000,
      "discount_percent": 0,
      "discount_amount": 0,
      "amount": 100000,
      "vat_rate": 10,
      "vat_amount": 10000,
      "total": 110000
    }
  ],
  "subtotal": 100000,
  "total_discount": 0,
  "total_vat": 10000,
  "total_amount": 110000,
  "currency": "VND",
  "payment_method": "string",
  "notes": "string"
}

Extract all visible information from the invoice image. For any text that appears blurry or unclear, make your best attempt to read it.`

const UserPromptOCRCorrection = `The following is OCR-extracted text from a Vietnamese invoice. It may contain errors.

OCR Text:
---
%s
---

Please:
1. Correct any obvious OCR errors (especially in Vietnamese diacritics)
2. Extract the structured invoice data

Output JSON with the same structure as before.`
