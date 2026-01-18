# Vietnam E-Invoice XML Template Reference

**Quick templates for all 5 provider formats with field mappings.**

---

## 1. TCT STANDARD TEMPLATE

```xml
<?xml version="1.0" encoding="UTF-8"?>
<Invoices>
  <Invoice>
    <!-- INVOICE HEADER -->
    <InvoiceNo>01</InvoiceNo>
    <InvoiceSeries>KK</InvoiceSeries>
    <InvoiceDate>2026-01-18</InvoiceDate>
    <Currency>VND</Currency>
    <ExchangeRate>1</ExchangeRate>

    <!-- SELLER INFORMATION (Người bán) -->
    <Seller>
      <Name>ABC Company Limited</Name>
      <Address>123 Le Loi Street, District 1, Ho Chi Minh City</Address>
      <TaxID>0123456789</TaxID>
      <PhoneNumber>+84-28-38000000</PhoneNumber>
      <Email>contact@abccompany.com</Email>
      <Website>www.abccompany.com</Website>
    </Seller>

    <!-- BUYER INFORMATION (Người mua) -->
    <Buyer>
      <Name>XYZ Corporation</Name>
      <Address>456 Nguyen Hue Boulevard, District 1, Ho Chi Minh City</Address>
      <TaxID>0987654321</TaxID>
      <PhoneNumber>+84-28-38111111</PhoneNumber>
      <Email>procurement@xyzcompany.com</Email>
    </Buyer>

    <!-- LINE ITEMS -->
    <Items>
      <Item>
        <ItemNo>1</ItemNo>
        <ItemCode>PROD-001</ItemCode>
        <ItemName>Industrial Equipment - Type A</ItemName>
        <Description>High-quality manufacturing equipment</Description>
        <UnitOfMeasure>piece</UnitOfMeasure>
        <Quantity>5</Quantity>
        <UnitPrice>500000</UnitPrice>
        <Amount>2500000</Amount>
        <TaxRatePercent>10</TaxRatePercent>
        <TaxAmount>250000</TaxAmount>
        <LineTotal>2750000</LineTotal>
      </Item>
      <Item>
        <ItemNo>2</ItemNo>
        <ItemCode>PROD-002</ItemCode>
        <ItemName>Raw Material - Grade B</ItemName>
        <Description>Premium quality raw materials</Description>
        <UnitOfMeasure>kg</UnitOfMeasure>
        <Quantity>100</Quantity>
        <UnitPrice>50000</UnitPrice>
        <Amount>5000000</Amount>
        <TaxRatePercent>5</TaxRatePercent>
        <TaxAmount>250000</TaxAmount>
        <LineTotal>5250000</LineTotal>
      </Item>
    </Items>

    <!-- TAX SUMMARY -->
    <SubtotalAmount>7500000</SubtotalAmount>
    <TaxAmount>500000</TaxAmount>
    <TotalAmount>8000000</TotalAmount>

    <!-- PAYMENT & NOTES -->
    <PaymentTerms>Net 30 days</PaymentTerms>
    <Remark>Thank you for your business. Payment by bank transfer.</Remark>

    <!-- DIGITAL SIGNATURE -->
    <Signature>
      <SignatureValue>MIIBkQIBAAJBALL...base64_encoded_signature...==</SignatureValue>
      <SignatureDate>2026-01-18T10:30:00+07:00</SignatureDate>
      <SignerName>Nguyen Van A</SignerName>
      <SignerPosition>Director</SignerPosition>
      <CertificateNo>CN=ABC Company,O=Tax Authority</CertificateNo>
    </Signature>
  </Invoice>
</Invoices>
```

---

## 2. VNPT ADAPTER TEMPLATE

```xml
<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <SubmitInvoiceRequest xmlns="http://vnpt.vn/invoice">
      <InvoiceData>
        <InvoiceNo>01</InvoiceNo>
        <InvoiceSeries>KK</InvoiceSeries>
        <InvoiceDate>2026-01-18</InvoiceDate>
        <SellerName>ABC Company Limited</SellerName>
        <SellerTaxCode>0123456789</SellerTaxCode>
        <SellerAddress>123 Le Loi Street, District 1, HCMC</SellerAddress>
        <BuyerName>XYZ Corporation</BuyerName>
        <BuyerTaxCode>0987654321</BuyerTaxCode>
        <BuyerAddress>456 Nguyen Hue, District 1, HCMC</BuyerAddress>

        <InvoiceDetails>
          <Detail>
            <DetailNo>1</DetailNo>
            <DetailName>Industrial Equipment - Type A</DetailName>
            <Quantity>5</Quantity>
            <UnitOfMeasure>piece</UnitOfMeasure>
            <Price>500000</Price>
            <Amount>2500000</Amount>
            <VAT>10</VAT>
            <VATAmount>250000</VATAmount>
          </Detail>
          <Detail>
            <DetailNo>2</DetailNo>
            <DetailName>Raw Material - Grade B</DetailName>
            <Quantity>100</Quantity>
            <UnitOfMeasure>kg</UnitOfMeasure>
            <Price>50000</Price>
            <Amount>5000000</Amount>
            <VAT>5</VAT>
            <VATAmount>250000</VATAmount>
          </Detail>
        </InvoiceDetails>

        <Total>8000000</Total>
        <Note>Thank you for your business</Note>
        <Signature>MIIBkQIBAAJBALL...base64_signature...==</Signature>
      </InvoiceData>
    </SubmitInvoiceRequest>
  </soap:Body>
</soap:Envelope>
```

---

## 3. MISA ADAPTER TEMPLATE

```xml
<?xml version="1.0" encoding="UTF-8"?>
<DonViTinhHD>
  <ThongTinChung>
    <SoHD>01</SoHD>
    <KyHieuHD>KK</KyHieuHD>
    <NgayHD>2026-01-18</NgayHD>
    <LoaiHD>Normal</LoaiHD>
  </ThongTinChung>

  <NguoiBan>
    <Ten>ABC Company Limited</Ten>
    <MST>0123456789</MST>
    <DiaChi>123 Le Loi Street, District 1, HCMC</DiaChi>
    <DienThoai>+84-28-38000000</DienThoai>
    <Email>contact@abccompany.com</Email>
  </NguoiBan>

  <NguoiMua>
    <Ten>XYZ Corporation</Ten>
    <MST>0987654321</MST>
    <DiaChi>456 Nguyen Hue, District 1, HCMC</DiaChi>
    <DienThoai>+84-28-38111111</DienThoai>
  </NguoiMua>

  <ChiTietHang>
    <Dong>
      <SoTT>1</SoTT>
      <MaHang>PROD-001</MaHang>
      <TenHang>Industrial Equipment - Type A</TenHang>
      <DonViTinh>piece</DonViTinh>
      <SoLuong>5</SoLuong>
      <GiaBan>500000</GiaBan>
      <ThanhTien>2500000</ThanhTien>
      <TyLeGiamGia>0</TyLeGiamGia>
      <TyLeVAT>10</TyLeVAT>
      <TienThueGTGT>250000</TienThueGTGT>
      <TongTien>2750000</TongTien>
    </Dong>
    <Dong>
      <SoTT>2</SoTT>
      <MaHang>PROD-002</MaHang>
      <TenHang>Raw Material - Grade B</TenHang>
      <DonViTinh>kg</DonViTinh>
      <SoLuong>100</SoLuong>
      <GiaBan>50000</GiaBan>
      <ThanhTien>5000000</ThanhTien>
      <TyLeGiamGia>0</TyLeGiamGia>
      <TyLeVAT>5</TyLeVAT>
      <TienThueGTGT>250000</TienThueGTGT>
      <TongTien>5250000</TongTien>
    </Dong>
  </ChiTietHang>

  <CongThuc>
    <CongTienHang>7500000</CongTienHang>
    <CongTienThue>500000</CongTienThue>
    <TongCong>8000000</TongCong>
  </CongThuc>

  <GhiChu>Thank you for your business</GhiChu>

  <QRCode>
    <Content>01|KK|2026-01-18|0123456789|0987654321|8000000|500000|XXXXXXXXXXX</Content>
    <Image>iVBORw0KGgoAAAANSUhEUgAAASwAAAEsCAIAAAD2HxkiAAAA...base64_image...==</Image>
  </QRCode>

  <KyDienTu>
    <GiaTriKy>MIIBkQIBAAJBALL...base64_signature...==</GiaTriKy>
    <ThoiGianKy>2026-01-18T10:30:00+07:00</ThoiGianKy>
    <TenNguoiKy>Nguyen Van A</TenNguoiKy>
    <ChucVuNguoiKy>Director</ChucVuNguoiKy>
  </KyDienTu>
</DonViTinhHD>
```

---

## 4. VIETTEL ADAPTER TEMPLATE

```xml
<?xml version="1.0" encoding="UTF-8"?>
<SInvoice>
  <Header>
    <No>01</No>
    <Series>KK</Series>
    <Date>2026-01-18</Date>
    <Type>Normal</Type>
    <Status>Active</Status>
  </Header>

  <Seller>
    <Name>ABC Company Limited</Name>
    <TaxNo>0123456789</TaxNo>
    <Address>123 Le Loi Street, District 1, HCMC</Address>
    <Phone>+84-28-38000000</Phone>
    <Email>contact@abccompany.com</Email>
  </Seller>

  <Buyer>
    <Name>XYZ Corporation</Name>
    <TaxNo>0987654321</TaxNo>
    <Address>456 Nguyen Hue, District 1, HCMC</Address>
    <Phone>+84-28-38111111</Phone>
  </Buyer>

  <Details>
    <Item>
      <No>1</No>
      <Code>PROD-001</Code>
      <Name>Industrial Equipment - Type A</Name>
      <Unit>piece</Unit>
      <Qty>5</Qty>
      <Price>500000</Price>
      <Amount>2500000</Amount>
      <Discount>0</Discount>
      <DiscountAmount>0</DiscountAmount>
      <TaxRate>10</TaxRate>
      <TaxAmount>250000</TaxAmount>
      <Total>2750000</Total>
    </Item>
    <Item>
      <No>2</No>
      <Code>PROD-002</Code>
      <Name>Raw Material - Grade B</Name>
      <Unit>kg</Unit>
      <Qty>100</Qty>
      <Price>50000</Price>
      <Amount>5000000</Amount>
      <Discount>0</Discount>
      <DiscountAmount>0</DiscountAmount>
      <TaxRate>5</TaxRate>
      <TaxAmount>250000</TaxAmount>
      <Total>5250000</Total>
    </Item>
  </Details>

  <Summary>
    <Subtotal>7500000</Subtotal>
    <TotalTax>500000</TotalTax>
    <GrandTotal>8000000</GrandTotal>
  </Summary>

  <Logistics>
    <TransportCost>0</TransportCost>
    <DeliveryAddress>456 Nguyen Hue, District 1, HCMC</DeliveryAddress>
    <DeliveryDate>2026-01-20</DeliveryDate>
  </Logistics>

  <Payment>
    <Method>Bank Transfer</Method>
    <Terms>Net 30 days</Terms>
    <BankAccount>0123456789</BankAccount>
    <BankName>Vietcombank</BankName>
  </Payment>

  <Notes>Thank you for your business</Notes>

  <Signature>
    <Value>MIIBkQIBAAJBALL...base64_signature...==</Value>
    <Date>2026-01-18T10:30:00+07:00</Date>
    <Signer>Nguyen Van A</Signer>
    <Position>Director</Position>
  </Signature>
</SInvoice>
```

---

## 5. FPT ADAPTER TEMPLATE

```xml
<?xml version="1.0" encoding="UTF-8"?>
<EInvoice>
  <InvoiceInfo>
    <No>01</No>
    <Series>KK</Series>
    <Date>2026-01-18</Date>
    <Type>Normal</Type>
    <Status>Approved</Status>
    <ReferenceNo></ReferenceNo>
    <ComplianceStatus>Compliant</ComplianceStatus>
    <UsageDeclaration>Normal</UsageDeclaration>
  </InvoiceInfo>

  <SellerInfo>
    <Name>ABC Company Limited</Name>
    <TaxID>0123456789</TaxID>
    <Address>123 Le Loi Street, District 1, HCMC</Address>
    <Phone>+84-28-38000000</Phone>
    <Email>contact@abccompany.com</Email>
    <Representative>Nguyen Van A</Representative>
  </SellerInfo>

  <BuyerInfo>
    <Name>XYZ Corporation</Name>
    <TaxID>0987654321</TaxID>
    <Address>456 Nguyen Hue, District 1, HCMC</Address>
    <Phone>+84-28-38111111</Phone>
    <Email>procurement@xyzcompany.com</Email>
  </BuyerInfo>

  <LineItems>
    <LineItem>
      <Seq>1</Seq>
      <Code>PROD-001</Code>
      <Description>Industrial Equipment - Type A</Description>
      <Unit>piece</Unit>
      <Quantity>5</Quantity>
      <Price>500000</Price>
      <NetAmount>2500000</NetAmount>
      <VATRate>10</VATRate>
      <VATAmount>250000</VATAmount>
      <Amount>2750000</Amount>
    </LineItem>
    <LineItem>
      <Seq>2</Seq>
      <Code>PROD-002</Code>
      <Description>Raw Material - Grade B</Description>
      <Unit>kg</Unit>
      <Quantity>100</Quantity>
      <Price>50000</Price>
      <NetAmount>5000000</NetAmount>
      <VATRate>5</VATRate>
      <VATAmount>250000</VATAmount>
      <Amount>5250000</Amount>
    </LineItem>
  </LineItems>

  <Summary>
    <SubTotal>7500000</SubTotal>
    <TotalVAT>500000</TotalVAT>
    <Amount>8000000</Amount>
  </Summary>

  <Remarks>Thank you for your business</Remarks>

  <Compliance>
    <Status>Compliant</Status>
    <ValidatedBy>FPT Tax System</ValidatedBy>
    <ValidationDate>2026-01-18T10:30:00+07:00</ValidationDate>
  </Compliance>

  <Signature>
    <SignatureValue>MIIBkQIBAAJBALL...base64_signature...==</SignatureValue>
    <SignatureDate>2026-01-18T10:30:00+07:00</SignatureDate>
    <SignerName>Nguyen Van A</SignerName>
    <SignerPosition>Director</SignerPosition>
    <CertSerial>1234567890ABCDEF</CertSerial>
  </Signature>
</EInvoice>
```

---

## QUICK FIELD REFERENCE TABLE

| Business Field | TCT | VNPT | MISA | Viettel | FPT |
|---|---|---|---|---|---|
| Company Name (Seller) | Seller/Name | SellerName | NguoiBan/Ten | Seller/Name | SellerInfo/Name |
| Tax ID (Seller) | Seller/TaxID | SellerTaxCode | NguoiBan/MST | Seller/TaxNo | SellerInfo/TaxID |
| Company Name (Buyer) | Buyer/Name | BuyerName | NguoiMua/Ten | Buyer/Name | BuyerInfo/Name |
| Tax ID (Buyer) | Buyer/TaxID | BuyerTaxCode | NguoiMua/MST | Buyer/TaxNo | BuyerInfo/TaxID |
| Product Name | ItemName | DetailName | TenHang | Name | Description |
| Quantity | Quantity | Quantity | SoLuong | Qty | Quantity |
| Unit Price | UnitPrice | Price | GiaBan | Price | Price |
| VAT Rate (%) | TaxRatePercent | VAT | TyLeVAT | TaxRate | VATRate |
| Line Total | LineTotal | Amount | TongTien | Total | Amount |
| Invoice Total | TotalAmount | Total | TongCong | GrandTotal | Amount |

---

## VALIDATION CHECKLIST

- [ ] Invoice number format: 1-6 digits
- [ ] Invoice series: 2-5 alphanumeric characters
- [ ] Tax IDs: 10 digits each
- [ ] Quantities: Positive decimals (2 decimal places)
- [ ] Prices: Positive integers (VND, no decimals)
- [ ] VAT rates: Only 0%, 5%, or 10%
- [ ] Date format: YYYY-MM-DD
- [ ] Decimal amounts: 2 places max
- [ ] Totals: amount + (amount * VAT rate)
- [ ] Signature: Valid base64, RSA-2048 + SHA-256

