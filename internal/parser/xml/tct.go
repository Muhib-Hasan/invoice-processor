package xml

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"time"

	"github.com/shopspring/decimal"

	"github.com/rezonia/invoice-processor/internal/model"
)

// TCT XML structures (Tax Authority standard format)
type tctInvoices struct {
	XMLName  xml.Name     `xml:"Invoices"`
	Invoices []tctInvoice `xml:"Invoice"`
}

type tctInvoice struct {
	XMLName        xml.Name  `xml:"Invoice"`
	InvoiceNo      string    `xml:"InvoiceNo"`
	InvoiceSeries  string    `xml:"InvoiceSeries"`
	InvoiceDate    string    `xml:"InvoiceDate"`
	InvoiceType    string    `xml:"InvoiceType"`
	Currency       string    `xml:"Currency"`
	ExchangeRate   string    `xml:"ExchangeRate"`
	Seller         tctParty  `xml:"Seller"`
	Buyer          tctParty  `xml:"Buyer"`
	Items          tctItems  `xml:"Items"`
	SubtotalAmount string    `xml:"SubtotalAmount"`
	TaxAmount      string    `xml:"TaxAmount"`
	TotalAmount    string    `xml:"TotalAmount"`
	PaymentTerms   string    `xml:"PaymentTerms"`
	Remarks        string    `xml:"Remarks"`
	Signature      *tctSig   `xml:"Signature"`
}

type tctParty struct {
	Name        string `xml:"Name"`
	Address     string `xml:"Address"`
	TaxID       string `xml:"TaxID"`
	PhoneNumber string `xml:"PhoneNumber"`
	Email       string `xml:"Email"`
	BankAccount string `xml:"BankAccount"`
	BankName    string `xml:"BankName"`
}

type tctItems struct {
	Items []tctItem `xml:"Item"`
}

type tctItem struct {
	ItemNo         int    `xml:"ItemNo"`
	ItemCode       string `xml:"ItemCode"`
	ItemName       string `xml:"ItemName"`
	Description    string `xml:"Description"`
	UnitOfMeasure  string `xml:"UnitOfMeasure"`
	Quantity       string `xml:"Quantity"`
	UnitPrice      string `xml:"UnitPrice"`
	Discount       string `xml:"Discount"`
	Amount         string `xml:"Amount"`
	TaxRatePercent int    `xml:"TaxRatePercent"`
	TaxAmount      string `xml:"TaxAmount"`
	LineTotal      string `xml:"LineTotal"`
}

type tctSig struct {
	SignatureValue string `xml:"SignatureValue"`
	SignatureDate  string `xml:"SignatureDate"`
	SignerName     string `xml:"SignerName"`
	SignerPosition string `xml:"SignerPosition"`
	CertificateNo  string `xml:"CertificateNo"`
}

// TCTAdapter parses TCT (Tax Authority) standard format
type TCTAdapter struct{}

// NewTCTAdapter creates a new TCT adapter
func NewTCTAdapter() *TCTAdapter {
	return &TCTAdapter{}
}

// Provider returns the provider type
func (a *TCTAdapter) Provider() model.Provider {
	return model.ProviderTCT
}

// CanParse checks if content is TCT format
func (a *TCTAdapter) CanParse(content []byte) bool {
	// Check for TCT-specific markers
	return bytes.Contains(content, []byte("<Invoice>")) &&
		bytes.Contains(content, []byte("<TaxID>")) &&
		!bytes.Contains(content, []byte("vnpt")) &&
		!bytes.Contains(content, []byte("<MST>")) &&
		!bytes.Contains(content, []byte("<SInvoice>"))
}

// Parse parses TCT XML into Invoice
func (a *TCTAdapter) Parse(ctx context.Context, r io.Reader) (*model.Invoice, error) {
	content, err := io.ReadAll(r)
	if err != nil {
		return nil, model.NewParseError(model.ProviderTCT, "content", "failed to read content", err)
	}

	// Try parsing as single invoice first
	var single tctInvoice
	if err := xml.Unmarshal(content, &single); err == nil && single.InvoiceNo != "" {
		return a.convertInvoice(&single, content)
	}

	// Try parsing as multiple invoices
	var multi tctInvoices
	if err := xml.Unmarshal(content, &multi); err != nil {
		return nil, model.NewParseError(model.ProviderTCT, "xml", "failed to parse XML", err)
	}

	if len(multi.Invoices) == 0 {
		return nil, model.NewParseError(model.ProviderTCT, "invoices", "no invoices found", nil)
	}

	// Return first invoice (caller can iterate if needed)
	return a.convertInvoice(&multi.Invoices[0], content)
}

func (a *TCTAdapter) convertInvoice(inv *tctInvoice, rawXML []byte) (*model.Invoice, error) {
	result := &model.Invoice{
		Number:   inv.InvoiceNo,
		Series:   inv.InvoiceSeries,
		Provider: model.ProviderTCT,
		Currency: inv.Currency,
		Remarks:  inv.Remarks,
		PaymentTerms: inv.PaymentTerms,
		RawXML:   rawXML,
	}

	// Parse date
	if date, err := parseDate(inv.InvoiceDate); err == nil {
		result.Date = date
	}

	// Parse invoice type
	result.Type = parseInvoiceType(inv.InvoiceType)

	// Parse exchange rate
	if rate, err := decimal.NewFromString(inv.ExchangeRate); err == nil {
		result.ExchangeRate = rate
	}

	// Convert parties
	result.Seller = convertTCTParty(inv.Seller)
	result.Buyer = convertTCTParty(inv.Buyer)

	// Convert line items
	for _, item := range inv.Items.Items {
		lineItem, err := convertTCTItem(item)
		if err != nil {
			return nil, err
		}
		result.Items = append(result.Items, *lineItem)
	}

	// Parse totals
	if amt, err := decimal.NewFromString(inv.SubtotalAmount); err == nil {
		result.SubtotalAmount = amt
	}
	if amt, err := decimal.NewFromString(inv.TaxAmount); err == nil {
		result.TaxAmount = amt
	}
	if amt, err := decimal.NewFromString(inv.TotalAmount); err == nil {
		result.TotalAmount = amt
	}

	// Convert signature
	if inv.Signature != nil {
		result.Signature = convertTCTSignature(inv.Signature)
	}

	return result, nil
}

func convertTCTParty(p tctParty) model.Party {
	return model.Party{
		Name:        p.Name,
		TaxID:       p.TaxID,
		Address:     p.Address,
		Phone:       p.PhoneNumber,
		Email:       p.Email,
		BankAccount: p.BankAccount,
		BankName:    p.BankName,
	}
}

func convertTCTItem(item tctItem) (*model.LineItem, error) {
	result := &model.LineItem{
		Number:      item.ItemNo,
		Code:        item.ItemCode,
		Name:        item.ItemName,
		Description: item.Description,
		Unit:        item.UnitOfMeasure,
		VATRate:     model.VATRate(item.TaxRatePercent),
	}

	// Parse decimal fields
	var err error
	if result.Quantity, err = decimal.NewFromString(item.Quantity); err != nil {
		result.Quantity = decimal.Zero
	}
	if result.UnitPrice, err = decimal.NewFromString(item.UnitPrice); err != nil {
		result.UnitPrice = decimal.Zero
	}
	if result.Discount, err = decimal.NewFromString(item.Discount); err != nil {
		result.Discount = decimal.Zero
	}
	if result.Amount, err = decimal.NewFromString(item.Amount); err != nil {
		result.Amount = decimal.Zero
	}
	if result.VATAmount, err = decimal.NewFromString(item.TaxAmount); err != nil {
		result.VATAmount = decimal.Zero
	}
	if result.Total, err = decimal.NewFromString(item.LineTotal); err != nil {
		result.Total = decimal.Zero
	}

	return result, nil
}

func convertTCTSignature(sig *tctSig) *model.Signature {
	result := &model.Signature{
		Value:          sig.SignatureValue,
		SignerName:     sig.SignerName,
		SignerPosition: sig.SignerPosition,
		CertSerial:     sig.CertificateNo,
	}

	if date, err := parseDate(sig.SignatureDate); err == nil {
		result.Date = date
	}

	return result
}

// Helper functions

func parseDate(s string) (time.Time, error) {
	formats := []string{
		"2006-01-02",
		"02/01/2006",
		"2006-01-02T15:04:05",
		"02/01/2006 15:04:05",
		time.RFC3339,
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("cannot parse date: %s", s)
}

func parseInvoiceType(s string) model.InvoiceType {
	switch s {
	case "Replacement", "replacement", "REPLACEMENT":
		return model.InvoiceTypeReplacement
	case "Adjustment", "adjustment", "ADJUSTMENT":
		return model.InvoiceTypeAdjustment
	default:
		return model.InvoiceTypeNormal
	}
}
