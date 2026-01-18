package xml

import (
	"bytes"
	"context"
	"encoding/xml"
	"io"

	"github.com/shopspring/decimal"

	"github.com/rezonia/invoice-processor/internal/model"
)

// Viettel XML structures (S-Invoice format)
// Supports both flat structure and TCT 2.0 nested structure (DLHDon/NDHDon)
type viettelInvoice struct {
	XMLName xml.Name `xml:"HDon"`

	// TCT 2.0 format: <HDon><DLHDon>...</DLHDon></HDon>
	DataLayer *viettelDataLayer `xml:"DLHDon"`

	// Legacy flat format: <HDon><TTChung>...</TTChung></HDon>
	InvoiceInfo    viettelInvoiceInfo `xml:"TTChung"`
	SellerInfo     viettelParty       `xml:"NBan"`
	BuyerInfo      viettelParty       `xml:"NMua"`
	Products       viettelProducts    `xml:"DSHHDVu"`
	Summary        viettelSummary     `xml:"TToan"`
	SignatureBlock *viettelSignBlock  `xml:"DSCKS"`
}

// viettelDataLayer handles TCT 2.0 format with DLHDon wrapper
type viettelDataLayer struct {
	InvoiceInfo    viettelInvoiceInfo    `xml:"TTChung"`
	InvoiceContent *viettelInvoiceContent `xml:"NDHDon"`
}

// viettelInvoiceContent handles NDHDon wrapper in TCT 2.0
type viettelInvoiceContent struct {
	SellerInfo viettelParty    `xml:"NBan"`
	BuyerInfo  viettelParty    `xml:"NMua"`
	Products   viettelProducts `xml:"DSHHDVu"`
	Summary    viettelSummary  `xml:"TToan"`
}

type viettelInvoiceInfo struct {
	KHMSHDon  string `xml:"KHMSHDon"` // Invoice series
	SHDon     string `xml:"SHDon"` // Invoice number
	NLap      string `xml:"NLap"` // Issue date
	LHDon     string `xml:"LHDon"` // Invoice type
	DVTTe     string `xml:"DVTTe"` // Currency
	TGia      string `xml:"TGia"` // Exchange rate
	HTTToan   string `xml:"HTTToan"` // Payment method
	THDon     string `xml:"THDon"` // Invoice status
	GChu      string `xml:"GChu"` // Notes
}

type viettelParty struct {
	MST    string `xml:"MST"` // Tax ID
	Ten    string `xml:"Ten"` // Name
	DChi   string `xml:"DChi"` // Address
	SDThoai string `xml:"SDThoai"` // Phone
	DCTDTu string `xml:"DCTDTu"` // Email
	STKNHang string `xml:"STKNHang"` // Bank account
	TNHang   string `xml:"TNHang"` // Bank name
}

type viettelProducts struct {
	Items []viettelItem `xml:"HHDVu"`
}

type viettelItem struct {
	STT      int    `xml:"STT"` // Line number
	MHHDVu   string `xml:"MHHDVu"` // Product code
	THHDVu   string `xml:"THHDVu"` // Product name
	DVTinh   string `xml:"DVTinh"` // Unit
	SLuong   string `xml:"SLuong"` // Quantity
	DGia     string `xml:"DGia"` // Unit price
	TLCKhau  string `xml:"TLCKhau"` // Discount rate
	STCKhau  string `xml:"STCKhau"` // Discount amount
	ThTien   string `xml:"ThTien"` // Amount before tax
	TSuat    string `xml:"TSuat"` // VAT rate
	TThue    string `xml:"TThue"` // VAT amount
	TgTToan  string `xml:"TgTToan"` // Line total
}

type viettelSummary struct {
	TgTCThue  string `xml:"TgTCThue"` // Total before tax
	TgTThue   string `xml:"TgTThue"` // Total VAT
	TgTTTBSo  string `xml:"TgTTTBSo"` // Total payment
	TgTTTBChu string `xml:"TgTTTBChu"` // Amount in words
}

type viettelSignBlock struct {
	Signatures []viettelSignature `xml:"CKS"`
}

type viettelSignature struct {
	GTCKy     string `xml:"GTCKy"` // Signature value
	NKy       string `xml:"NKy"` // Sign date
	TNguoiKy  string `xml:"TNguoiKy"` // Signer name
	CDanhKy   string `xml:"CDanhKy"` // Signer position
	SHCThu    string `xml:"SHCThu"` // Certificate serial
}

// ViettelAdapter parses Viettel S-Invoice format
type ViettelAdapter struct{}

// NewViettelAdapter creates a new Viettel adapter
func NewViettelAdapter() *ViettelAdapter {
	return &ViettelAdapter{}
}

// Provider returns the provider type
func (a *ViettelAdapter) Provider() model.Provider {
	return model.ProviderViettel
}

// CanParse checks if content is Viettel format
func (a *ViettelAdapter) CanParse(content []byte) bool {
	// Viettel uses <HDon> root and Vietnamese abbreviated tags
	return bytes.Contains(content, []byte("<HDon>")) ||
		bytes.Contains(content, []byte("<KHMSHDon>")) ||
		bytes.Contains(content, []byte("viettel")) ||
		bytes.Contains(content, []byte("sinvoice"))
}

// Parse parses Viettel XML into Invoice
func (a *ViettelAdapter) Parse(ctx context.Context, r io.Reader) (*model.Invoice, error) {
	content, err := io.ReadAll(r)
	if err != nil {
		return nil, model.NewParseError(model.ProviderViettel, "content", "failed to read content", err)
	}

	var inv viettelInvoice
	if err := xml.Unmarshal(content, &inv); err != nil {
		return nil, model.NewParseError(model.ProviderViettel, "xml", "failed to parse XML", err)
	}

	return a.convertInvoice(&inv, content)
}

func (a *ViettelAdapter) convertInvoice(inv *viettelInvoice, rawXML []byte) (*model.Invoice, error) {
	// Determine which format we're dealing with
	var invoiceInfo viettelInvoiceInfo
	var sellerInfo, buyerInfo viettelParty
	var products viettelProducts
	var summary viettelSummary

	if inv.DataLayer != nil {
		// TCT 2.0 format: data is inside DLHDon/NDHDon wrappers
		invoiceInfo = inv.DataLayer.InvoiceInfo
		if inv.DataLayer.InvoiceContent != nil {
			sellerInfo = inv.DataLayer.InvoiceContent.SellerInfo
			buyerInfo = inv.DataLayer.InvoiceContent.BuyerInfo
			products = inv.DataLayer.InvoiceContent.Products
			summary = inv.DataLayer.InvoiceContent.Summary
		}
	} else {
		// Legacy flat format
		invoiceInfo = inv.InvoiceInfo
		sellerInfo = inv.SellerInfo
		buyerInfo = inv.BuyerInfo
		products = inv.Products
		summary = inv.Summary
	}

	result := &model.Invoice{
		Number:   invoiceInfo.SHDon,
		Series:   invoiceInfo.KHMSHDon,
		Provider: model.ProviderViettel,
		Currency: invoiceInfo.DVTTe,
		Remarks:  invoiceInfo.GChu,
		RawXML:   rawXML,
	}

	// Parse date
	if date, err := parseDate(invoiceInfo.NLap); err == nil {
		result.Date = date
	}

	// Parse invoice type
	result.Type = parseInvoiceType(invoiceInfo.LHDon)

	// Parse exchange rate
	if rate, err := decimal.NewFromString(invoiceInfo.TGia); err == nil {
		result.ExchangeRate = rate
	}

	// Convert parties
	result.Seller = model.Party{
		Name:        sellerInfo.Ten,
		TaxID:       sellerInfo.MST,
		Address:     sellerInfo.DChi,
		Phone:       sellerInfo.SDThoai,
		Email:       sellerInfo.DCTDTu,
		BankAccount: sellerInfo.STKNHang,
		BankName:    sellerInfo.TNHang,
	}

	result.Buyer = model.Party{
		Name:        buyerInfo.Ten,
		TaxID:       buyerInfo.MST,
		Address:     buyerInfo.DChi,
		Phone:       buyerInfo.SDThoai,
		Email:       buyerInfo.DCTDTu,
		BankAccount: buyerInfo.STKNHang,
		BankName:    buyerInfo.TNHang,
	}

	// Convert line items
	for _, item := range products.Items {
		lineItem := convertViettelItem(item)
		result.Items = append(result.Items, *lineItem)
	}

	// Parse totals
	if amt, err := decimal.NewFromString(summary.TgTCThue); err == nil {
		result.SubtotalAmount = amt
	}
	if amt, err := decimal.NewFromString(summary.TgTThue); err == nil {
		result.TaxAmount = amt
	}
	if amt, err := decimal.NewFromString(summary.TgTTTBSo); err == nil {
		result.TotalAmount = amt
	}

	// Convert signature (take first if available)
	if inv.SignatureBlock != nil && len(inv.SignatureBlock.Signatures) > 0 {
		result.Signature = convertViettelSignature(&inv.SignatureBlock.Signatures[0])
	}

	return result, nil
}

func convertViettelItem(item viettelItem) *model.LineItem {
	result := &model.LineItem{
		Number: item.STT,
		Code:   item.MHHDVu,
		Name:   item.THHDVu,
		Unit:   item.DVTinh,
	}

	// Parse VAT rate (may be percentage string like "10" or "10%")
	if rate, err := decimal.NewFromString(item.TSuat); err == nil {
		result.VATRate = model.VATRate(rate.IntPart())
	}

	// Parse decimal fields
	if qty, err := decimal.NewFromString(item.SLuong); err == nil {
		result.Quantity = qty
	}
	if price, err := decimal.NewFromString(item.DGia); err == nil {
		result.UnitPrice = price
	}
	if disc, err := decimal.NewFromString(item.TLCKhau); err == nil {
		result.Discount = disc
	}
	if discAmt, err := decimal.NewFromString(item.STCKhau); err == nil {
		result.DiscountAmt = discAmt
	}
	if amt, err := decimal.NewFromString(item.ThTien); err == nil {
		result.Amount = amt
	}
	if vat, err := decimal.NewFromString(item.TThue); err == nil {
		result.VATAmount = vat
	}
	if total, err := decimal.NewFromString(item.TgTToan); err == nil {
		result.Total = total
	}

	return result
}

func convertViettelSignature(sig *viettelSignature) *model.Signature {
	result := &model.Signature{
		Value:          sig.GTCKy,
		SignerName:     sig.TNguoiKy,
		SignerPosition: sig.CDanhKy,
		CertSerial:     sig.SHCThu,
	}

	if date, err := parseDate(sig.NKy); err == nil {
		result.Date = date
	}

	return result
}
