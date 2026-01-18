package model_test

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rezonia/invoice-processor/internal/model"
)

func TestInvoice_Creation(t *testing.T) {
	inv := model.Invoice{
		Number:   "01",
		Series:   "KK",
		Date:     time.Date(2026, 1, 18, 0, 0, 0, 0, time.UTC),
		Type:     model.InvoiceTypeNormal,
		Provider: model.ProviderVNPT,
		Seller: model.Party{
			Name:  "ABC Company",
			TaxID: "0123456789",
		},
		Buyer: model.Party{
			Name:  "XYZ Corp",
			TaxID: "9876543210",
		},
		Currency: "VND",
	}

	assert.Equal(t, "01", inv.Number)
	assert.Equal(t, "KK", inv.Series)
	assert.Equal(t, model.ProviderVNPT, inv.Provider)
	assert.Equal(t, model.InvoiceTypeNormal, inv.Type)
	assert.Equal(t, "0123456789", inv.Seller.TaxID)
	assert.Equal(t, "9876543210", inv.Buyer.TaxID)
	assert.Equal(t, "VND", inv.Currency)
}

func TestLineItem_Calculate(t *testing.T) {
	item := model.LineItem{
		Number:    1,
		Name:      "Product A",
		Unit:      "piece",
		Quantity:  decimal.NewFromInt(10),
		UnitPrice: decimal.NewFromInt(100000),
		VATRate:   model.VATRate10,
	}

	item.Calculate()

	// Amount = 10 * 100000 = 1,000,000
	assert.True(t, item.Amount.Equal(decimal.NewFromInt(1000000)),
		"Expected amount 1000000, got %s", item.Amount.String())

	// No discount
	assert.True(t, item.DiscountAmt.IsZero())

	// VAT = 1,000,000 * 10% = 100,000
	assert.True(t, item.VATAmount.Equal(decimal.NewFromInt(100000)),
		"Expected VAT 100000, got %s", item.VATAmount.String())

	// Total = 1,000,000 + 100,000 = 1,100,000
	assert.True(t, item.Total.Equal(decimal.NewFromInt(1100000)),
		"Expected total 1100000, got %s", item.Total.String())
}

func TestLineItem_CalculateWithDiscount(t *testing.T) {
	item := model.LineItem{
		Number:    1,
		Name:      "Product B",
		Unit:      "piece",
		Quantity:  decimal.NewFromInt(5),
		UnitPrice: decimal.NewFromInt(200000),
		Discount:  decimal.NewFromInt(10), // 10% discount
		VATRate:   model.VATRate10,
	}

	item.Calculate()

	// Amount = 5 * 200,000 = 1,000,000
	assert.True(t, item.Amount.Equal(decimal.NewFromInt(1000000)))

	// Discount = 1,000,000 * 10% = 100,000
	assert.True(t, item.DiscountAmt.Equal(decimal.NewFromInt(100000)))

	// Taxable = 1,000,000 - 100,000 = 900,000
	// VAT = 900,000 * 10% = 90,000
	assert.True(t, item.VATAmount.Equal(decimal.NewFromInt(90000)),
		"Expected VAT 90000, got %s", item.VATAmount.String())

	// Total = 900,000 + 90,000 = 990,000
	assert.True(t, item.Total.Equal(decimal.NewFromInt(990000)),
		"Expected total 990000, got %s", item.Total.String())
}

func TestInvoice_CalculateTotals(t *testing.T) {
	inv := model.Invoice{
		Items: []model.LineItem{
			{
				Name:      "Item 1",
				Quantity:  decimal.NewFromInt(2),
				UnitPrice: decimal.NewFromInt(100000),
				VATRate:   model.VATRate10,
			},
			{
				Name:      "Item 2",
				Quantity:  decimal.NewFromInt(3),
				UnitPrice: decimal.NewFromInt(50000),
				VATRate:   model.VATRate5,
			},
		},
	}

	inv.CalculateTotals()

	// Item 1: Amount=200,000, VAT=20,000
	// Item 2: Amount=150,000, VAT=7,500
	// Subtotal = 350,000
	// TaxAmount = 27,500
	// Total = 377,500

	assert.True(t, inv.SubtotalAmount.Equal(decimal.NewFromInt(350000)),
		"Expected subtotal 350000, got %s", inv.SubtotalAmount.String())

	assert.True(t, inv.TaxAmount.Equal(decimal.NewFromInt(27500)),
		"Expected tax 27500, got %s", inv.TaxAmount.String())

	assert.True(t, inv.TotalAmount.Equal(decimal.NewFromInt(377500)),
		"Expected total 377500, got %s", inv.TotalAmount.String())
}

func TestProviderConstants(t *testing.T) {
	providers := []model.Provider{
		model.ProviderTCT,
		model.ProviderVNPT,
		model.ProviderMISA,
		model.ProviderViettel,
		model.ProviderFPT,
	}

	for _, p := range providers {
		assert.NotEmpty(t, string(p))
	}
}

func TestVATRates(t *testing.T) {
	assert.Equal(t, 0, int(model.VATRate0))
	assert.Equal(t, 5, int(model.VATRate5))
	assert.Equal(t, 10, int(model.VATRate10))
}

func TestParseError(t *testing.T) {
	err := &model.ParseError{
		Provider: model.ProviderMISA,
		Field:    "TaxID",
		Message:  "invalid format",
	}

	require.Contains(t, err.Error(), "MISA")
	require.Contains(t, err.Error(), "TaxID")
	require.Contains(t, err.Error(), "invalid format")
}

func TestParseError_WithCause(t *testing.T) {
	cause := assert.AnError
	err := model.NewParseError(model.ProviderVNPT, "Date", "parse failed", cause)

	require.Contains(t, err.Error(), "VNPT")
	require.Contains(t, err.Error(), "Date")
	require.ErrorIs(t, err, cause)
}

func TestValidationError(t *testing.T) {
	err := model.NewValidationError("TaxID", "12345", "length", "must be 10 digits")

	require.Contains(t, err.Error(), "TaxID")
	require.Contains(t, err.Error(), "12345")
	require.Contains(t, err.Error(), "10 digits")
}
