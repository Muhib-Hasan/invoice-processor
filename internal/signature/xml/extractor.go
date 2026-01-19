package xml

import (
	"bytes"
	"fmt"

	"github.com/beevik/etree"
)

// XML namespaces
const (
	XMLDSigNamespace = "http://www.w3.org/2000/09/xmldsig#"
)

// SignatureExtractor extracts XMLDSig signature elements from different XML formats
type SignatureExtractor struct{}

// NewSignatureExtractor creates a new signature extractor
func NewSignatureExtractor() *SignatureExtractor {
	return &SignatureExtractor{}
}

// ExtractionResult contains the extracted signature and related elements
type ExtractionResult struct {
	// SignatureElement is the <Signature> element
	SignatureElement *etree.Element
	// SignedElement is the element that was signed (parent of signature or referenced element)
	SignedElement *etree.Element
	// Document is the parsed XML document
	Document *etree.Document
	// Provider indicates which provider format was detected
	Provider string
}

// Extract finds and extracts the XMLDSig signature from XML data
func (e *SignatureExtractor) Extract(data []byte) (*ExtractionResult, error) {
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(data); err != nil {
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}

	root := doc.Root()
	if root == nil {
		return nil, fmt.Errorf("empty XML document")
	}

	// Try to find Signature element
	sig := findSignatureElement(root)
	if sig == nil {
		return nil, fmt.Errorf("no Signature element found in document")
	}

	// Determine signed element (usually the document root or signature parent)
	signedElement := sig.Parent()
	if signedElement == nil {
		signedElement = root
	}

	provider := detectProvider(root)

	return &ExtractionResult{
		SignatureElement: sig,
		SignedElement:    signedElement,
		Document:         doc,
		Provider:         provider,
	}, nil
}

// findSignatureElement searches for the Signature element in the document
func findSignatureElement(root *etree.Element) *etree.Element {
	// Search paths for different providers
	searchPaths := []string{
		// Direct child
		"Signature",
		// With namespace prefix
		"ds:Signature",
		// TCT format
		"Invoice/Signature",
		"Invoices/Invoice/Signature",
		// VNPT format
		"SInvoice/Signature",
		// MISA format
		"HoaDon/Signature",
		// Viettel format
		"HDon/TTChung/TTKhac/Signature",
		"HDon/Signature",
		// FPT format
		"EInvoice/Signature",
	}

	// First try direct path searches
	for _, path := range searchPaths {
		if elem := root.FindElement(path); elem != nil {
			return elem
		}
	}

	// Fallback: recursive search for any element named "Signature"
	return findElementRecursive(root, "Signature")
}

// findElementRecursive searches for an element by local name recursively
func findElementRecursive(elem *etree.Element, localName string) *etree.Element {
	// Check if current element matches (ignoring namespace prefix)
	if elem.Tag == localName || hasLocalName(elem, localName) {
		return elem
	}

	// Search children
	for _, child := range elem.ChildElements() {
		if found := findElementRecursive(child, localName); found != nil {
			return found
		}
	}

	return nil
}

// hasLocalName checks if element has the given local name (ignoring namespace prefix)
func hasLocalName(elem *etree.Element, localName string) bool {
	tag := elem.Tag
	// Handle prefixed tags like "ds:Signature"
	if idx := bytes.IndexByte([]byte(tag), ':'); idx >= 0 {
		tag = tag[idx+1:]
	}
	return tag == localName
}

// detectProvider identifies the invoice provider from XML structure
func detectProvider(root *etree.Element) string {
	switch root.Tag {
	case "SInvoice":
		return "VNPT"
	case "HDon":
		return "Viettel"
	case "EInvoice":
		return "FPT"
	case "HoaDon":
		return "MISA"
	case "Invoice", "Invoices":
		return "TCT"
	default:
		return "Unknown"
	}
}

// ExtractCertificateData extracts the base64-encoded certificate from a Signature element
func ExtractCertificateData(sig *etree.Element) ([]byte, error) {
	// Path: Signature/KeyInfo/X509Data/X509Certificate
	paths := []string{
		"KeyInfo/X509Data/X509Certificate",
		"ds:KeyInfo/ds:X509Data/ds:X509Certificate",
	}

	for _, path := range paths {
		if certElem := sig.FindElement(path); certElem != nil {
			certText := certElem.Text()
			if certText != "" {
				return []byte(certText), nil
			}
		}
	}

	return nil, fmt.Errorf("no X509Certificate found in Signature")
}

// CanExtract returns true if the data appears to be XML with a signature
func (e *SignatureExtractor) CanExtract(data []byte) bool {
	// Quick check for XML
	if len(data) < 5 {
		return false
	}

	// Check for XML declaration or root element
	trimmed := bytes.TrimSpace(data)
	if !bytes.HasPrefix(trimmed, []byte("<?xml")) && !bytes.HasPrefix(trimmed, []byte("<")) {
		return false
	}

	// Check for Signature element
	return bytes.Contains(data, []byte("<Signature")) ||
		bytes.Contains(data, []byte("<ds:Signature")) ||
		bytes.Contains(data, []byte(":Signature"))
}
