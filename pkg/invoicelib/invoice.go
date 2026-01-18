// Package invoicelib provides a public API for processing Vietnam e-invoices.
//
// This package exposes the core types and interfaces for parsing, validating,
// and extracting data from Vietnam e-invoices in XML and PDF formats.
//
// Example usage:
//
//	parser := invoicelib.NewParser()
//	invoice, err := parser.ParseXML(ctx, reader)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(invoice.TotalAmount)
package invoicelib

import "github.com/rezonia/invoice-processor/internal/model"

// Re-export core types for public API
type (
	Invoice     = model.Invoice
	LineItem    = model.LineItem
	Party       = model.Party
	Signature   = model.Signature
	Provider    = model.Provider
	VATRate     = model.VATRate
	InvoiceType = model.InvoiceType
)

// Re-export provider constants
const (
	ProviderTCT     = model.ProviderTCT
	ProviderVNPT    = model.ProviderVNPT
	ProviderMISA    = model.ProviderMISA
	ProviderViettel = model.ProviderViettel
	ProviderFPT     = model.ProviderFPT
	ProviderUnknown = model.ProviderUnknown
)

// Re-export VAT rates
const (
	VATRate0  = model.VATRate0
	VATRate5  = model.VATRate5
	VATRate10 = model.VATRate10
)

// Re-export invoice types
const (
	InvoiceTypeNormal      = model.InvoiceTypeNormal
	InvoiceTypeReplacement = model.InvoiceTypeReplacement
	InvoiceTypeAdjustment  = model.InvoiceTypeAdjustment
)

// Re-export error types
type (
	ParseError      = model.ParseError
	ValidationError = model.ValidationError
	ExtractionError = model.ExtractionError
)
