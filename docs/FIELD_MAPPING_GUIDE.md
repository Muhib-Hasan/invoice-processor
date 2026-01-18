# Vietnam E-Invoice Field Mapping Guide

**Universal mapping between standard invoice data and provider-specific XML formats.**

---

## UNIVERSAL INVOICE DATA MODEL

```typescript
{
  // INVOICE METADATA
  invoiceNumber: string;              // "01"
  invoiceSeries: string;              // "KK"
  invoiceDate: string;                // "2026-01-18"
  invoiceType: "Normal" | "Replacement" | "Adjustment";

  // SELLER DATA
  sellerName: string;
  sellerTaxId: string;                // 10 digits: "0123456789"
  sellerAddress: string;
  sellerPhone?: string;
  sellerEmail?: string;

  // BUYER DATA
  buyerName: string;
  buyerTaxId: string;
  buyerAddress: string;
  buyerPhone?: string;
  buyerEmail?: string;

  // LINE ITEMS
  items: Array<{
    itemNo: number;
    itemCode?: string;
    itemName: string;
    unitOfMeasure: string;
    quantity: number;
    unitPrice: number;
    discountPercent?: number;
    vatRate: 0 | 5 | 10;
  }>;

  // TOTALS (auto-calculated)
  subtotal: number;                   // VND
  totalVat: number;                   // VND
  totalAmount: number;                // VND

  // METADATA
  paymentTerms?: string;
  remarks?: string;
  currency: "VND";
}
```

---

## PROVIDER-SPECIFIC MAPPING

### TCT Standard (Tax Authority Format)

| Universal Field | TCT XML Path | Type | Required | Notes |
|---|---|---|---|---|
| invoiceNumber | Invoice/InvoiceNo | string | Yes | 1-6 digits |
| invoiceSeries | Invoice/InvoiceSeries | string | Yes | 2-5 alphanumeric |
| invoiceDate | Invoice/InvoiceDate | date | Yes | YYYY-MM-DD |
| sellerName | Invoice/Seller/Name | string | Yes | Max 255 chars |
| sellerTaxId | Invoice/Seller/TaxID | string | Yes | 10 digits |
| sellerAddress | Invoice/Seller/Address | string | Yes | Max 255 chars |
| sellerPhone | Invoice/Seller/PhoneNumber | string | No | Include +84 prefix |
| sellerEmail | Invoice/Seller/Email | email | No | Valid email format |
| buyerName | Invoice/Buyer/Name | string | Yes | Max 255 chars |
| buyerTaxId | Invoice/Buyer/TaxID | string | Yes | 10 digits |
| buyerAddress | Invoice/Buyer/Address | string | Yes | Max 255 chars |
| buyerPhone | Invoice/Buyer/PhoneNumber | string | No | Include +84 prefix |
| buyerEmail | Invoice/Buyer/Email | email | No | Valid email format |
| itemNo | Invoice/Items/Item/ItemNo | int | Yes | Sequential 1+ |
| itemCode | Invoice/Items/Item/ItemCode | string | No | Max 50 chars |
| itemName | Invoice/Items/Item/ItemName | string | Yes | Max 255 chars |
| unitOfMeasure | Invoice/Items/Item/UnitOfMeasure | string | Yes | "piece", "kg", "meter", "hour", "set" |
| quantity | Invoice/Items/Item/Quantity | decimal | Yes | 2 decimal places, >0 |
| unitPrice | Invoice/Items/Item/UnitPrice | int | Yes | VND, no decimals, >0 |
| vatRate | Invoice/Items/Item/TaxRatePercent | enum | Yes | 0, 5, or 10 |
| subtotal | Invoice/SubtotalAmount | int | Yes | Sum of (qty * unitPrice) |
| totalVat | Invoice/TaxAmount | int | Yes | Sum of line VAT amounts |
| totalAmount | Invoice/TotalAmount | int | Yes | subtotal + totalVat |
| remarks | Invoice/Remark | string | No | Max 500 chars |

---

### VNPT E-Invoice Format

| Universal Field | VNPT XML Path | Type | Required | Mapping Notes |
|---|---|---|---|---|
| invoiceNumber | InvoiceData/InvoiceNo | string | Yes | Direct mapping |
| invoiceSeries | InvoiceData/InvoiceSeries | string | Yes | Direct mapping |
| invoiceDate | InvoiceData/InvoiceDate | date | Yes | Format: YYYY-MM-DD |
| sellerName | InvoiceData/SellerName | string | Yes | Single field (no separate Name/Address) |
| sellerTaxId | InvoiceData/SellerTaxCode | string | Yes | Renamed: TaxCode not TaxID |
| sellerAddress | InvoiceData/SellerAddress | string | Yes | Separate from SellerName |
| buyerName | InvoiceData/BuyerName | string | Yes | Single field structure |
| buyerTaxId | InvoiceData/BuyerTaxCode | string | Yes | Renamed: TaxCode |
| buyerAddress | InvoiceData/BuyerAddress | string | Yes | Separate field |
| itemNo | InvoiceData/InvoiceDetails/Detail/DetailNo | int | Yes | Sequential numbering |
| itemName | InvoiceData/InvoiceDetails/Detail/DetailName | string | Yes | No itemCode field |
| quantity | InvoiceData/InvoiceDetails/Detail/Quantity | decimal | Yes | 2 decimal places |
| unitOfMeasure | InvoiceData/InvoiceDetails/Detail/UnitOfMeasure | string | Yes | Same as TCT |
| unitPrice | InvoiceData/InvoiceDetails/Detail/Price | int | Yes | Decimal places allowed |
| amount | InvoiceData/InvoiceDetails/Detail/Amount | int | Yes | qty * unitPrice |
| vatRate | InvoiceData/InvoiceDetails/Detail/VAT | enum | Yes | Values: 0, 5, 10 |
| vatAmount | InvoiceData/InvoiceDetails/Detail/VATAmount | int | Yes | amount * (VAT/100) |
| totalAmount | InvoiceData/Total | int | Yes | Sum of all amounts + VAT |

**VNPT-Specific Rules:**
- No direct seller/buyer address separation (combined string)
- No itemCode field available
- VAT auto-calculated by VNPT system
- SOAP wrapper required (xmlns namespace)
- Quantity supports decimals unlike TCT

---

### MISA meInvoice Format

| Universal Field | MISA XML Path | Type | Required | Mapping Notes |
|---|---|---|---|---|
| invoiceNumber | DonViTinhHD/ThongTinChung/SoHD | string | Yes | Direct mapping |
| invoiceSeries | DonViTinhHD/ThongTinChung/KyHieuHD | string | Yes | Vietnamese field name |
| invoiceDate | DonViTinhHD/ThongTinChung/NgayHD | date | Yes | Format: YYYY-MM-DD |
| invoiceType | DonViTinhHD/ThongTinChung/LoaiHD | enum | No | "Normal", "Replacement" |
| sellerName | DonViTinhHD/NguoiBan/Ten | string | Yes | Vietnamese: "Ten" = Name |
| sellerTaxId | DonViTinhHD/NguoiBan/MST | string | Yes | Vietnamese: "MST" = Tax ID |
| sellerAddress | DonViTinhHD/NguoiBan/DiaChi | string | Yes | Vietnamese: "DiaChi" = Address |
| sellerPhone | DonViTinhHD/NguoiBan/DienThoai | string | No | Vietnamese field names |
| sellerEmail | DonViTinhHD/NguoiBan/Email | email | No | Optional |
| buyerName | DonViTinhHD/NguoiMua/Ten | string | Yes | "Nguoi Mua" = Buyer |
| buyerTaxId | DonViTinhHD/NguoiMua/MST | string | Yes | Tax ID field |
| buyerAddress | DonViTinhHD/NguoiMua/DiaChi | string | Yes | Address field |
| itemNo | DonViTinhHD/ChiTietHang/Dong/SoTT | int | Yes | "SoTT" = Sequence |
| itemCode | DonViTinhHD/ChiTietHang/Dong/MaHang | string | No | "MaHang" = Item Code |
| itemName | DonViTinhHD/ChiTietHang/Dong/TenHang | string | Yes | "TenHang" = Item Name |
| unitOfMeasure | DonViTinhHD/ChiTietHang/Dong/DonViTinh | string | Yes | "DonViTinh" = Unit |
| quantity | DonViTinhHD/ChiTietHang/Dong/SoLuong | decimal | Yes | "SoLuong" = Quantity |
| unitPrice | DonViTinhHD/ChiTietHang/Dong/GiaBan | decimal | Yes | "GiaBan" = Selling Price |
| amount | DonViTinhHD/ChiTietHang/Dong/ThanhTien | int | Yes | "ThanhTien" = Amount |
| discountPercent | DonViTinhHD/ChiTietHang/Dong/TyLeGiamGia | decimal | No | "TyLeGiamGia" = Discount % |
| vatRate | DonViTinhHD/ChiTietHang/Dong/TyLeVAT | enum | Yes | "TyLeVAT" = VAT Rate |
| vatAmount | DonViTinhHD/ChiTietHang/Dong/TienThueGTGT | int | Yes | "GTGT" = VAT |
| subtotal | DonViTinhHD/CongThuc/CongTienHang | int | Yes | Formula: sum items |
| totalVat | DonViTinhHD/CongThuc/CongTienThue | int | Yes | Formula: sum VAT |
| totalAmount | DonViTinhHD/CongThuc/TongCong | int | Yes | Formula: total |
| remarks | DonViTinhHD/GhiChu | string | No | Optional notes |

**MISA-Specific Rules:**
- 100% Vietnamese field naming convention
- Supports QR code generation (separate QRCode element)
- Automatic formula calculation: Totals auto-computed
- Supports batch invoice submission
- Discount percentage field available

---

### Viettel S-Invoice Format

| Universal Field | Viettel XML Path | Type | Required | Mapping Notes |
|---|---|---|---|---|
| invoiceNumber | SInvoice/Header/No | string | Yes | Direct |
| invoiceSeries | SInvoice/Header/Series | string | Yes | Direct |
| invoiceDate | SInvoice/Header/Date | date | Yes | YYYY-MM-DD |
| invoiceType | SInvoice/Header/Type | enum | Yes | "Normal", "Replacement" |
| sellerName | SInvoice/Seller/Name | string | Yes | Direct |
| sellerTaxId | SInvoice/Seller/TaxNo | string | Yes | "TaxNo" vs others' "TaxID" |
| sellerAddress | SInvoice/Seller/Address | string | Yes | Direct |
| sellerPhone | SInvoice/Seller/Phone | string | No | Optional |
| sellerEmail | SInvoice/Seller/Email | email | No | Optional |
| buyerName | SInvoice/Buyer/Name | string | Yes | Direct |
| buyerTaxId | SInvoice/Buyer/TaxNo | string | Yes | Same field naming as seller |
| buyerAddress | SInvoice/Buyer/Address | string | Yes | Direct |
| itemNo | SInvoice/Details/Item/No | int | Yes | Sequential |
| itemCode | SInvoice/Details/Item/Code | string | No | "Code" (shorter) |
| itemName | SInvoice/Details/Item/Name | string | Yes | Direct |
| unitOfMeasure | SInvoice/Details/Item/Unit | string | Yes | "Unit" (shorter) |
| quantity | SInvoice/Details/Item/Qty | decimal | Yes | "Qty" abbreviation |
| unitPrice | SInvoice/Details/Item/Price | decimal | Yes | Supports decimals |
| amount | SInvoice/Details/Item/Amount | int | Yes | qty * price |
| discountPercent | SInvoice/Details/Item/Discount | decimal | No | Discount % |
| discountAmount | SInvoice/Details/Item/DiscountAmount | int | No | Calculated discount |
| vatRate | SInvoice/Details/Item/TaxRate | enum | Yes | "TaxRate" |
| vatAmount | SInvoice/Details/Item/TaxAmount | int | Yes | amount * (rate/100) |
| itemTotal | SInvoice/Details/Item/Total | int | Yes | amount + VAT |
| subtotal | SInvoice/Summary/Subtotal | int | Yes | Direct |
| totalVat | SInvoice/Summary/TotalTax | int | Yes | "TotalTax" |
| totalAmount | SInvoice/Summary/GrandTotal | int | Yes | Direct |

**Viettel-Specific Rules:**
- Supports delivery logistics (TransportCost, DeliveryAddress, DeliveryDate)
- Payment method tracking (Payment/Method, Payment/Terms)
- Discount field available at line-item level
- Freight/shipping cost support

---

### FPT eInvoice Format

| Universal Field | FPT XML Path | Type | Required | Mapping Notes |
|---|---|---|---|---|
| invoiceNumber | EInvoice/InvoiceInfo/No | string | Yes | Direct |
| invoiceSeries | EInvoice/InvoiceInfo/Series | string | Yes | Direct |
| invoiceDate | EInvoice/InvoiceInfo/Date | date | Yes | YYYY-MM-DD |
| invoiceType | EInvoice/InvoiceInfo/Type | enum | Yes | "Normal", "Adjustment" |
| invoiceStatus | EInvoice/InvoiceInfo/Status | enum | Yes | "Approved", "Rejected" |
| complianceStatus | EInvoice/InvoiceInfo/ComplianceStatus | enum | No | "Compliant", "Non-Compliant" |
| sellerName | EInvoice/SellerInfo/Name | string | Yes | Direct |
| sellerTaxId | EInvoice/SellerInfo/TaxID | string | Yes | Direct |
| sellerAddress | EInvoice/SellerInfo/Address | string | Yes | Direct |
| sellerPhone | EInvoice/SellerInfo/Phone | string | No | Optional |
| sellerEmail | EInvoice/SellerInfo/Email | email | No | Optional |
| sellerRep | EInvoice/SellerInfo/Representative | string | No | Contact person |
| buyerName | EInvoice/BuyerInfo/Name | string | Yes | Direct |
| buyerTaxId | EInvoice/BuyerInfo/TaxID | string | Yes | Direct |
| buyerAddress | EInvoice/BuyerInfo/Address | string | Yes | Direct |
| buyerPhone | EInvoice/BuyerInfo/Phone | string | No | Optional |
| buyerEmail | EInvoice/BuyerInfo/Email | email | No | Optional |
| itemSeq | EInvoice/LineItems/LineItem/Seq | int | Yes | Sequential |
| itemCode | EInvoice/LineItems/LineItem/Code | string | No | Optional |
| itemDesc | EInvoice/LineItems/LineItem/Description | string | Yes | Item description |
| unitOfMeasure | EInvoice/LineItems/LineItem/Unit | string | Yes | Unit of measure |
| quantity | EInvoice/LineItems/LineItem/Quantity | decimal | Yes | Qty with decimals |
| unitPrice | EInvoice/LineItems/LineItem/Price | decimal | Yes | Supports decimals |
| netAmount | EInvoice/LineItems/LineItem/NetAmount | int | Yes | Before VAT |
| vatRate | EInvoice/LineItems/LineItem/VATRate | enum | Yes | 0, 5, 10 |
| vatAmount | EInvoice/LineItems/LineItem/VATAmount | int | Yes | Calculated VAT |
| lineAmount | EInvoice/LineItems/LineItem/Amount | int | Yes | netAmount + VAT |
| subtotal | EInvoice/Summary/SubTotal | int | Yes | Sum before VAT |
| totalVat | EInvoice/Summary/TotalVAT | int | Yes | Sum of VAT |
| totalAmount | EInvoice/Summary/Amount | int | Yes | Grand total |
| remarks | EInvoice/Remarks | string | No | Optional notes |

**FPT-Specific Rules:**
- Compliance status tracking (compliance validation)
- Usage declaration field (UsageDeclaration: "Normal", "Adjustment")
- Reference number for replacements/adjustments
- Validation timestamp and validator name
- Certificate serial tracking for audit trail

---

## CONVERSION ALGORITHM

```typescript
function convertInvoice(universal: UniversalInvoice, targetProvider: string): string {
  switch (targetProvider) {
    case 'TCT':
      return convertToTCT(universal);
    case 'VNPT':
      return convertToVNPT(universal);
    case 'MISA':
      return convertToMISA(universal);
    case 'VIETTEL':
      return convertToVIETTEL(universal);
    case 'FPT':
      return convertToFPT(universal);
  }
}

// Example TCT conversion
function convertToTCT(u: UniversalInvoice): string {
  const items = u.items.map((item, idx) => ({
    ItemNo: idx + 1,
    ItemCode: item.itemCode || '',
    ItemName: item.itemName,
    UnitOfMeasure: item.unitOfMeasure,
    Quantity: item.quantity,
    UnitPrice: item.unitPrice,
    Amount: item.quantity * item.unitPrice,
    TaxRatePercent: item.vatRate,
    TaxAmount: (item.quantity * item.unitPrice * item.vatRate) / 100,
    LineTotal: item.quantity * item.unitPrice +
               ((item.quantity * item.unitPrice * item.vatRate) / 100)
  }));

  return buildXML('Invoices/Invoice', {
    InvoiceNo: u.invoiceNumber,
    InvoiceSeries: u.invoiceSeries,
    InvoiceDate: u.invoiceDate,
    // ... etc
  });
}
```

---

## VALIDATION RULES BY PROVIDER

### Universal Rules (All Providers)
- [ ] Invoice number: 1-6 digits
- [ ] Invoice series: 2-5 alphanumeric characters
- [ ] Tax IDs: Exactly 10 digits
- [ ] Invoice date: Valid YYYY-MM-DD
- [ ] Quantities: 0 < qty ≤ 999,999.99
- [ ] Unit prices: 0 < price ≤ 999,999,999 (VND)
- [ ] VAT rates: Only 0, 5, or 10
- [ ] Seller ≠ Buyer (different tax IDs)
- [ ] At least 1 line item

### Provider-Specific Rules
**MISA:** QR code generation required
**Viettel:** TransportCost validation (≥ 0)
**FPT:** ComplianceStatus must be provided
**VNPT:** Batch size limit ≤ 100 invoices per submission

