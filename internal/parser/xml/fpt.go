package xml

import (
	"bytes"
	"context"
	"encoding/xml"
	"io"

	"github.com/shopspring/decimal"

	"github.com/rezonia/invoice-processor/internal/model"
)

// FPT XML structures (eInvoice format)
type fptInvoice struct {
	XMLName     xml.Name        `xml:"EInvoice"`
	Header      fptHeader       `xml:"Header"`
	Seller      fptCompany      `xml:"Seller"`
	Buyer       fptCompany      `xml:"Buyer"`
	Details     fptDetails      `xml:"Details"`
	Totals      fptTotals       `xml:"Totals"`
	Signatures  *fptSignatures  `xml:"Signatures"`
}

type fptHeader struct {
	InvoiceNumber  string `xml:"InvoiceNumber"`
	InvoiceSeries  string `xml:"InvoiceSeries"`
	InvoiceDate    string `xml:"InvoiceDate"`
	InvoiceType    string `xml:"InvoiceType"`
	CurrencyCode   string `xml:"CurrencyCode"`
	ExchangeRate   string `xml:"ExchangeRate"`
	PaymentMethod  string `xml:"PaymentMethod"`
	DueDate        string `xml:"DueDate"`
	Notes          string `xml:"Notes"`
}

type fptCompany struct {
	CompanyCode    string `xml:"CompanyCode"`
	CompanyName    string `xml:"CompanyName"`
	TaxCode        string `xml:"TaxCode"`
	Address        string `xml:"Address"`
	PhoneNumber    string `xml:"PhoneNumber"`
	EmailAddress   string `xml:"EmailAddress"`
	BankAccountNo  string `xml:"BankAccountNo"`
	BankName       string `xml:"BankName"`
	ContactPerson  string `xml:"ContactPerson"`
}

type fptDetails struct {
	Lines []fptLine `xml:"Line"`
}

type fptLine struct {
	LineNumber     int    `xml:"LineNumber"`
	ProductCode    string `xml:"ProductCode"`
	ProductName    string `xml:"ProductName"`
	ProductDesc    string `xml:"ProductDesc"`
	UnitOfMeasure  string `xml:"UnitOfMeasure"`
	Quantity       string `xml:"Quantity"`
	UnitPrice      string `xml:"UnitPrice"`
	DiscountRate   string `xml:"DiscountRate"`
	DiscountAmount string `xml:"DiscountAmount"`
	LineAmount     string `xml:"LineAmount"`
	VATRatePercent string `xml:"VATRatePercent"`
	VATAmount      string `xml:"VATAmount"`
	LineTotal      string `xml:"LineTotal"`
}

type fptTotals struct {
	SubTotal        string `xml:"SubTotal"`
	TotalDiscount   string `xml:"TotalDiscount"`
	TotalVAT        string `xml:"TotalVAT"`
	GrandTotal      string `xml:"GrandTotal"`
	AmountInWords   string `xml:"AmountInWords"`
}

type fptSignatures struct {
	SellerSignature *fptSignature `xml:"SellerSignature"`
	BuyerSignature  *fptSignature `xml:"BuyerSignature"`
}

type fptSignature struct {
	SignatureValue   string `xml:"SignatureValue"`
	SignedDateTime   string `xml:"SignedDateTime"`
	SignerFullName   string `xml:"SignerFullName"`
	SignerJobTitle   string `xml:"SignerJobTitle"`
	CertificateSerial string `xml:"CertificateSerial"`
}

// FPTAdapter parses FPT eInvoice format
type FPTAdapter struct{}

// NewFPTAdapter creates a new FPT adapter
func NewFPTAdapter() *FPTAdapter {
	return &FPTAdapter{}
}

// Provider returns the provider type
func (a *FPTAdapter) Provider() model.Provider {
	return model.ProviderFPT
}

// CanParse checks if content is FPT format
func (a *FPTAdapter) CanParse(content []byte) bool {
	// FPT uses <EInvoice> root element
	return bytes.Contains(content, []byte("<EInvoice>")) ||
		bytes.Contains(content, []byte("fpt")) ||
		bytes.Contains(content, []byte("FPT"))
}

// Parse parses FPT XML into Invoice
func (a *FPTAdapter) Parse(ctx context.Context, r io.Reader) (*model.Invoice, error) {
	content, err := io.ReadAll(r)
	if err != nil {
		return nil, model.NewParseError(model.ProviderFPT, "content", "failed to read content", err)
	}

	var inv fptInvoice
	if err := xml.Unmarshal(content, &inv); err != nil {
		return nil, model.NewParseError(model.ProviderFPT, "xml", "failed to parse XML", err)
	}

	return a.convertInvoice(&inv, content)
}

func (a *FPTAdapter) convertInvoice(inv *fptInvoice, rawXML []byte) (*model.Invoice, error) {
	result := &model.Invoice{
		Number:   inv.Header.InvoiceNumber,
		Series:   inv.Header.InvoiceSeries,
		Provider: model.ProviderFPT,
		Currency: inv.Header.CurrencyCode,
		Remarks:  inv.Header.Notes,
		RawXML:   rawXML,
	}

	// Parse date
	if date, err := parseDate(inv.Header.InvoiceDate); err == nil {
		result.Date = date
	}

	// Parse invoice type
	result.Type = parseInvoiceType(inv.Header.InvoiceType)

	// Parse exchange rate
	if rate, err := decimal.NewFromString(inv.Header.ExchangeRate); err == nil {
		result.ExchangeRate = rate
	}

	// Convert parties
	result.Seller = model.Party{
		Name:        inv.Seller.CompanyName,
		TaxID:       inv.Seller.TaxCode,
		Address:     inv.Seller.Address,
		Phone:       inv.Seller.PhoneNumber,
		Email:       inv.Seller.EmailAddress,
		BankAccount: inv.Seller.BankAccountNo,
		BankName:    inv.Seller.BankName,
	}

	result.Buyer = model.Party{
		Name:        inv.Buyer.CompanyName,
		TaxID:       inv.Buyer.TaxCode,
		Address:     inv.Buyer.Address,
		Phone:       inv.Buyer.PhoneNumber,
		Email:       inv.Buyer.EmailAddress,
		BankAccount: inv.Buyer.BankAccountNo,
		BankName:    inv.Buyer.BankName,
	}

	// Convert line items
	for _, line := range inv.Details.Lines {
		lineItem := convertFPTLine(line)
		result.Items = append(result.Items, *lineItem)
	}

	// Parse totals
	if amt, err := decimal.NewFromString(inv.Totals.SubTotal); err == nil {
		result.SubtotalAmount = amt
	}
	if amt, err := decimal.NewFromString(inv.Totals.TotalVAT); err == nil {
		result.TaxAmount = amt
	}
	if amt, err := decimal.NewFromString(inv.Totals.GrandTotal); err == nil {
		result.TotalAmount = amt
	}

	// Convert seller signature (primary)
	if inv.Signatures != nil && inv.Signatures.SellerSignature != nil {
		result.Signature = convertFPTSignature(inv.Signatures.SellerSignature)
	}

	return result, nil
}

func convertFPTLine(line fptLine) *model.LineItem {
	result := &model.LineItem{
		Number:      line.LineNumber,
		Code:        line.ProductCode,
		Name:        line.ProductName,
		Description: line.ProductDesc,
		Unit:        line.UnitOfMeasure,
	}

	// Parse VAT rate
	if rate, err := decimal.NewFromString(line.VATRatePercent); err == nil {
		result.VATRate = model.VATRate(rate.IntPart())
	}

	// Parse decimal fields
	if qty, err := decimal.NewFromString(line.Quantity); err == nil {
		result.Quantity = qty
	}
	if price, err := decimal.NewFromString(line.UnitPrice); err == nil {
		result.UnitPrice = price
	}
	if disc, err := decimal.NewFromString(line.DiscountRate); err == nil {
		result.Discount = disc
	}
	if discAmt, err := decimal.NewFromString(line.DiscountAmount); err == nil {
		result.DiscountAmt = discAmt
	}
	if amt, err := decimal.NewFromString(line.LineAmount); err == nil {
		result.Amount = amt
	}
	if vat, err := decimal.NewFromString(line.VATAmount); err == nil {
		result.VATAmount = vat
	}
	if total, err := decimal.NewFromString(line.LineTotal); err == nil {
		result.Total = total
	}

	return result
}

func convertFPTSignature(sig *fptSignature) *model.Signature {
	result := &model.Signature{
		Value:          sig.SignatureValue,
		SignerName:     sig.SignerFullName,
		SignerPosition: sig.SignerJobTitle,
		CertSerial:     sig.CertificateSerial,
	}

	if date, err := parseDate(sig.SignedDateTime); err == nil {
		result.Date = date
	}

	return result
}
