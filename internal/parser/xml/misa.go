package xml

import (
	"bytes"
	"context"
	"encoding/xml"
	"io"

	"github.com/shopspring/decimal"

	"github.com/rezonia/invoice-processor/internal/model"
)

// MISA XML structures
type misaInvoice struct {
	XMLName       xml.Name        `xml:"Invoice"`
	InvoiceData   misaInvoiceData `xml:"InvoiceData"`
	SellerInfo    misaParty       `xml:"SellerInfo"`
	BuyerInfo     misaParty       `xml:"BuyerInfo"`
	InvoiceDetail misaDetails     `xml:"InvoiceDetail"`
	TotalSection  misaTotals      `xml:"TotalSection"`
	SignatureInfo *misaSignature  `xml:"SignatureInfo"`
}

type misaInvoiceData struct {
	InvoiceNumber string `xml:"InvoiceNumber"`
	InvoiceSeries string `xml:"InvoiceSeries"`
	InvoiceDate   string `xml:"InvoiceDate"`
	InvoiceType   string `xml:"InvoiceType"`
	CurrencyCode  string `xml:"CurrencyCode"`
	ExchangeRate  string `xml:"ExchangeRate"`
	PaymentMethod string `xml:"PaymentMethod"`
	PaymentTerms  string `xml:"PaymentTerms"`
	Description   string `xml:"Description"`
}

type misaParty struct {
	MST         string `xml:"MST"` // Tax ID
	CompanyName string `xml:"CompanyName"`
	Address     string `xml:"Address"`
	Phone       string `xml:"Phone"`
	Email       string `xml:"Email"`
	BankAccount string `xml:"BankAccount"`
	BankName    string `xml:"BankName"`
}

type misaDetails struct {
	Items []misaItem `xml:"Item"`
}

type misaItem struct {
	STT         int    `xml:"STT"` // Line number
	MaHang      string `xml:"MaHang"` // Item code
	TenHang     string `xml:"TenHang"` // Item name
	MoTa        string `xml:"MoTa"` // Description
	DVT         string `xml:"DVT"` // Unit
	SoLuong     string `xml:"SoLuong"` // Quantity
	DonGia      string `xml:"DonGia"` // Unit price
	ChietKhau   string `xml:"ChietKhau"` // Discount %
	TienCK      string `xml:"TienCK"` // Discount amount
	ThanhTien   string `xml:"ThanhTien"` // Amount
	ThueSuat    string `xml:"ThueSuat"` // VAT rate
	TienThue    string `xml:"TienThue"` // VAT amount
	TongCong    string `xml:"TongCong"` // Total
}

type misaTotals struct {
	TongTienHang    string `xml:"TongTienHang"` // Subtotal
	TongChietKhau   string `xml:"TongChietKhau"` // Total discount
	TongTienThue    string `xml:"TongTienThue"` // Total VAT
	TongThanhToan   string `xml:"TongThanhToan"` // Total payment
	SoTienBangChu   string `xml:"SoTienBangChu"` // Amount in words
}

type misaSignature struct {
	GiaTriChuKy   string `xml:"GiaTriChuKy"` // Signature value
	NgayKy        string `xml:"NgayKy"` // Signature date
	NguoiKy       string `xml:"NguoiKy"` // Signer name
	ChucDanh      string `xml:"ChucDanh"` // Signer title
	SoChungThu    string `xml:"SoChungThu"` // Certificate serial
}

// MISAAdapter parses MISA invoice format
type MISAAdapter struct{}

// NewMISAAdapter creates a new MISA adapter
func NewMISAAdapter() *MISAAdapter {
	return &MISAAdapter{}
}

// Provider returns the provider type
func (a *MISAAdapter) Provider() model.Provider {
	return model.ProviderMISA
}

// CanParse checks if content is MISA format
func (a *MISAAdapter) CanParse(content []byte) bool {
	// MISA uses <MST> for tax ID and Vietnamese field names
	return bytes.Contains(content, []byte("<MST>")) ||
		bytes.Contains(content, []byte("<TenHang>")) ||
		bytes.Contains(content, []byte("MISA")) ||
		bytes.Contains(content, []byte("misa"))
}

// Parse parses MISA XML into Invoice
func (a *MISAAdapter) Parse(ctx context.Context, r io.Reader) (*model.Invoice, error) {
	content, err := io.ReadAll(r)
	if err != nil {
		return nil, model.NewParseError(model.ProviderMISA, "content", "failed to read content", err)
	}

	var inv misaInvoice
	if err := xml.Unmarshal(content, &inv); err != nil {
		return nil, model.NewParseError(model.ProviderMISA, "xml", "failed to parse XML", err)
	}

	return a.convertInvoice(&inv, content)
}

func (a *MISAAdapter) convertInvoice(inv *misaInvoice, rawXML []byte) (*model.Invoice, error) {
	result := &model.Invoice{
		Number:       inv.InvoiceData.InvoiceNumber,
		Series:       inv.InvoiceData.InvoiceSeries,
		Provider:     model.ProviderMISA,
		Currency:     inv.InvoiceData.CurrencyCode,
		Remarks:      inv.InvoiceData.Description,
		PaymentTerms: inv.InvoiceData.PaymentTerms,
		RawXML:       rawXML,
	}

	// Parse date
	if date, err := parseDate(inv.InvoiceData.InvoiceDate); err == nil {
		result.Date = date
	}

	// Parse invoice type
	result.Type = parseInvoiceType(inv.InvoiceData.InvoiceType)

	// Parse exchange rate
	if rate, err := decimal.NewFromString(inv.InvoiceData.ExchangeRate); err == nil {
		result.ExchangeRate = rate
	}

	// Convert parties
	result.Seller = model.Party{
		Name:        inv.SellerInfo.CompanyName,
		TaxID:       inv.SellerInfo.MST,
		Address:     inv.SellerInfo.Address,
		Phone:       inv.SellerInfo.Phone,
		Email:       inv.SellerInfo.Email,
		BankAccount: inv.SellerInfo.BankAccount,
		BankName:    inv.SellerInfo.BankName,
	}

	result.Buyer = model.Party{
		Name:        inv.BuyerInfo.CompanyName,
		TaxID:       inv.BuyerInfo.MST,
		Address:     inv.BuyerInfo.Address,
		Phone:       inv.BuyerInfo.Phone,
		Email:       inv.BuyerInfo.Email,
		BankAccount: inv.BuyerInfo.BankAccount,
		BankName:    inv.BuyerInfo.BankName,
	}

	// Convert line items
	for _, item := range inv.InvoiceDetail.Items {
		lineItem := convertMISAItem(item)
		result.Items = append(result.Items, *lineItem)
	}

	// Parse totals
	if amt, err := decimal.NewFromString(inv.TotalSection.TongTienHang); err == nil {
		result.SubtotalAmount = amt
	}
	if amt, err := decimal.NewFromString(inv.TotalSection.TongTienThue); err == nil {
		result.TaxAmount = amt
	}
	if amt, err := decimal.NewFromString(inv.TotalSection.TongThanhToan); err == nil {
		result.TotalAmount = amt
	}

	// Convert signature
	if inv.SignatureInfo != nil {
		result.Signature = convertMISASignature(inv.SignatureInfo)
	}

	return result, nil
}

func convertMISAItem(item misaItem) *model.LineItem {
	result := &model.LineItem{
		Number:      item.STT,
		Code:        item.MaHang,
		Name:        item.TenHang,
		Description: item.MoTa,
		Unit:        item.DVT,
	}

	// Parse VAT rate
	if rate, err := decimal.NewFromString(item.ThueSuat); err == nil {
		result.VATRate = model.VATRate(rate.IntPart())
	}

	// Parse decimal fields
	if qty, err := decimal.NewFromString(item.SoLuong); err == nil {
		result.Quantity = qty
	}
	if price, err := decimal.NewFromString(item.DonGia); err == nil {
		result.UnitPrice = price
	}
	if disc, err := decimal.NewFromString(item.ChietKhau); err == nil {
		result.Discount = disc
	}
	if discAmt, err := decimal.NewFromString(item.TienCK); err == nil {
		result.DiscountAmt = discAmt
	}
	if amt, err := decimal.NewFromString(item.ThanhTien); err == nil {
		result.Amount = amt
	}
	if vat, err := decimal.NewFromString(item.TienThue); err == nil {
		result.VATAmount = vat
	}
	if total, err := decimal.NewFromString(item.TongCong); err == nil {
		result.Total = total
	}

	return result
}

func convertMISASignature(sig *misaSignature) *model.Signature {
	result := &model.Signature{
		Value:          sig.GiaTriChuKy,
		SignerName:     sig.NguoiKy,
		SignerPosition: sig.ChucDanh,
		CertSerial:     sig.SoChungThu,
	}

	if date, err := parseDate(sig.NgayKy); err == nil {
		result.Date = date
	}

	return result
}
