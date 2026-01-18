package xml

import (
	"bytes"
	"context"
	"encoding/xml"
	"io"

	"github.com/shopspring/decimal"

	"github.com/rezonia/invoice-processor/internal/model"
)

// VNPT XML structures
type vnptInvoice struct {
	XMLName        xml.Name     `xml:"SInvoice"`
	InvoiceNo      string       `xml:"InvoiceNo"`
	InvoiceSeries  string       `xml:"InvoiceSeries"`
	InvoiceDate    string       `xml:"InvoiceDate"`
	InvoiceType    string       `xml:"InvoiceType"`
	Currency       string       `xml:"Currency"`
	ExchangeRate   string       `xml:"ExchangeRate"`
	Seller         vnptSeller   `xml:"Seller"`
	Buyer          vnptBuyer    `xml:"Buyer"`
	Products       vnptProducts `xml:"Products"`
	Summary        vnptSummary  `xml:"Summary"`
	PaymentMethod  string       `xml:"PaymentMethod"`
	PaymentTerms   string       `xml:"PaymentTerms"`
	Note           string       `xml:"Note"`
	SignInfo       *vnptSign    `xml:"SignInfo"`
}

type vnptSeller struct {
	SellerName      string `xml:"SellerName"`
	SellerTaxCode   string `xml:"SellerTaxCode"`
	SellerAddress   string `xml:"SellerAddress"`
	SellerPhone     string `xml:"SellerPhone"`
	SellerEmail     string `xml:"SellerEmail"`
	SellerBankAcc   string `xml:"SellerBankAcc"`
	SellerBankName  string `xml:"SellerBankName"`
}

type vnptBuyer struct {
	BuyerName      string `xml:"BuyerName"`
	BuyerTaxCode   string `xml:"BuyerTaxCode"`
	BuyerAddress   string `xml:"BuyerAddress"`
	BuyerPhone     string `xml:"BuyerPhone"`
	BuyerEmail     string `xml:"BuyerEmail"`
	BuyerBankAcc   string `xml:"BuyerBankAcc"`
	BuyerBankName  string `xml:"BuyerBankName"`
}

type vnptProducts struct {
	Products []vnptProduct `xml:"Product"`
}

type vnptProduct struct {
	LineNo       int    `xml:"LineNo"`
	ProdCode     string `xml:"ProdCode"`
	ProdName     string `xml:"ProdName"`
	ProdUnit     string `xml:"ProdUnit"`
	ProdQuantity string `xml:"ProdQuantity"`
	ProdPrice    string `xml:"ProdPrice"`
	Discount     string `xml:"Discount"`
	DiscountAmt  string `xml:"DiscountAmt"`
	Amount       string `xml:"Amount"`
	VATRate      string `xml:"VATRate"`
	VATAmount    string `xml:"VATAmount"`
	Total        string `xml:"Total"`
}

type vnptSummary struct {
	TotalAmount    string `xml:"TotalAmount"`
	TotalDiscount  string `xml:"TotalDiscount"`
	TotalVATAmount string `xml:"TotalVATAmount"`
	TotalPayment   string `xml:"TotalPayment"`
	AmountInWords  string `xml:"AmountInWords"`
}

type vnptSign struct {
	SignatureValue string `xml:"SignatureValue"`
	SignedDate     string `xml:"SignedDate"`
	SignerName     string `xml:"SignerName"`
	SignerTitle    string `xml:"SignerTitle"`
	CertSerial     string `xml:"CertSerial"`
}

// VNPTAdapter parses VNPT invoice format
type VNPTAdapter struct{}

// NewVNPTAdapter creates a new VNPT adapter
func NewVNPTAdapter() *VNPTAdapter {
	return &VNPTAdapter{}
}

// Provider returns the provider type
func (a *VNPTAdapter) Provider() model.Provider {
	return model.ProviderVNPT
}

// CanParse checks if content is VNPT format
func (a *VNPTAdapter) CanParse(content []byte) bool {
	// VNPT uses <SInvoice> root element
	return bytes.Contains(content, []byte("<SInvoice>")) ||
		bytes.Contains(content, []byte("vnpt")) ||
		bytes.Contains(content, []byte("VNPT"))
}

// Parse parses VNPT XML into Invoice
func (a *VNPTAdapter) Parse(ctx context.Context, r io.Reader) (*model.Invoice, error) {
	content, err := io.ReadAll(r)
	if err != nil {
		return nil, model.NewParseError(model.ProviderVNPT, "content", "failed to read content", err)
	}

	var inv vnptInvoice
	if err := xml.Unmarshal(content, &inv); err != nil {
		return nil, model.NewParseError(model.ProviderVNPT, "xml", "failed to parse XML", err)
	}

	return a.convertInvoice(&inv, content)
}

func (a *VNPTAdapter) convertInvoice(inv *vnptInvoice, rawXML []byte) (*model.Invoice, error) {
	result := &model.Invoice{
		Number:       inv.InvoiceNo,
		Series:       inv.InvoiceSeries,
		Provider:     model.ProviderVNPT,
		Currency:     inv.Currency,
		Remarks:      inv.Note,
		PaymentTerms: inv.PaymentTerms,
		RawXML:       rawXML,
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
	result.Seller = model.Party{
		Name:        inv.Seller.SellerName,
		TaxID:       inv.Seller.SellerTaxCode,
		Address:     inv.Seller.SellerAddress,
		Phone:       inv.Seller.SellerPhone,
		Email:       inv.Seller.SellerEmail,
		BankAccount: inv.Seller.SellerBankAcc,
		BankName:    inv.Seller.SellerBankName,
	}

	result.Buyer = model.Party{
		Name:        inv.Buyer.BuyerName,
		TaxID:       inv.Buyer.BuyerTaxCode,
		Address:     inv.Buyer.BuyerAddress,
		Phone:       inv.Buyer.BuyerPhone,
		Email:       inv.Buyer.BuyerEmail,
		BankAccount: inv.Buyer.BuyerBankAcc,
		BankName:    inv.Buyer.BuyerBankName,
	}

	// Convert line items
	for _, prod := range inv.Products.Products {
		lineItem := convertVNPTProduct(prod)
		result.Items = append(result.Items, *lineItem)
	}

	// Parse totals
	if amt, err := decimal.NewFromString(inv.Summary.TotalAmount); err == nil {
		result.SubtotalAmount = amt
	}
	if amt, err := decimal.NewFromString(inv.Summary.TotalVATAmount); err == nil {
		result.TaxAmount = amt
	}
	if amt, err := decimal.NewFromString(inv.Summary.TotalPayment); err == nil {
		result.TotalAmount = amt
	}

	// Convert signature
	if inv.SignInfo != nil {
		result.Signature = convertVNPTSignature(inv.SignInfo)
	}

	return result, nil
}

func convertVNPTProduct(prod vnptProduct) *model.LineItem {
	result := &model.LineItem{
		Number:      prod.LineNo,
		Code:        prod.ProdCode,
		Name:        prod.ProdName,
		Unit:        prod.ProdUnit,
	}

	// Parse VAT rate
	if rate, err := decimal.NewFromString(prod.VATRate); err == nil {
		result.VATRate = model.VATRate(rate.IntPart())
	}

	// Parse decimal fields
	if qty, err := decimal.NewFromString(prod.ProdQuantity); err == nil {
		result.Quantity = qty
	}
	if price, err := decimal.NewFromString(prod.ProdPrice); err == nil {
		result.UnitPrice = price
	}
	if disc, err := decimal.NewFromString(prod.Discount); err == nil {
		result.Discount = disc
	}
	if discAmt, err := decimal.NewFromString(prod.DiscountAmt); err == nil {
		result.DiscountAmt = discAmt
	}
	if amt, err := decimal.NewFromString(prod.Amount); err == nil {
		result.Amount = amt
	}
	if vat, err := decimal.NewFromString(prod.VATAmount); err == nil {
		result.VATAmount = vat
	}
	if total, err := decimal.NewFromString(prod.Total); err == nil {
		result.Total = total
	}

	return result
}

func convertVNPTSignature(sig *vnptSign) *model.Signature {
	result := &model.Signature{
		Value:          sig.SignatureValue,
		SignerName:     sig.SignerName,
		SignerPosition: sig.SignerTitle,
		CertSerial:     sig.CertSerial,
	}

	if date, err := parseDate(sig.SignedDate); err == nil {
		result.Date = date
	}

	return result
}
