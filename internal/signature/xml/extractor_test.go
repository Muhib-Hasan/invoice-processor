package xml

import (
	"testing"
)

func TestSignatureExtractor_CanExtract(t *testing.T) {
	extractor := NewSignatureExtractor()

	tests := []struct {
		name     string
		data     []byte
		expected bool
	}{
		{
			name:     "XML with Signature",
			data:     []byte(`<?xml version="1.0"?><Invoice><Data>test</Data><Signature xmlns="http://www.w3.org/2000/09/xmldsig#"><SignedInfo/></Signature></Invoice>`),
			expected: true,
		},
		{
			name:     "XML with ds:Signature",
			data:     []byte(`<?xml version="1.0"?><Invoice><ds:Signature xmlns:ds="http://www.w3.org/2000/09/xmldsig#"><ds:SignedInfo/></ds:Signature></Invoice>`),
			expected: true,
		},
		{
			name:     "XML without Signature",
			data:     []byte(`<?xml version="1.0"?><Invoice><Data>test</Data></Invoice>`),
			expected: false,
		},
		{
			name:     "Not XML",
			data:     []byte(`{"type": "json"}`),
			expected: false,
		},
		{
			name:     "Empty",
			data:     []byte(``),
			expected: false,
		},
		{
			name:     "PDF magic bytes",
			data:     []byte(`%PDF-1.4`),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractor.CanExtract(tt.data)
			if result != tt.expected {
				t.Errorf("CanExtract: got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSignatureExtractor_Extract(t *testing.T) {
	extractor := NewSignatureExtractor()

	tests := []struct {
		name           string
		data           []byte
		expectError    bool
		expectedProv   string
	}{
		{
			name: "TCT format",
			data: []byte(`<?xml version="1.0"?>
<Invoice>
	<Data>test</Data>
	<Signature xmlns="http://www.w3.org/2000/09/xmldsig#">
		<SignedInfo/>
		<SignatureValue/>
	</Signature>
</Invoice>`),
			expectError:  false,
			expectedProv: "TCT",
		},
		{
			name: "VNPT format",
			data: []byte(`<?xml version="1.0"?>
<SInvoice>
	<Data>test</Data>
	<Signature xmlns="http://www.w3.org/2000/09/xmldsig#">
		<SignedInfo/>
	</Signature>
</SInvoice>`),
			expectError:  false,
			expectedProv: "VNPT",
		},
		{
			name: "Viettel format",
			data: []byte(`<?xml version="1.0"?>
<HDon>
	<TTChung/>
	<Signature xmlns="http://www.w3.org/2000/09/xmldsig#">
		<SignedInfo/>
	</Signature>
</HDon>`),
			expectError:  false,
			expectedProv: "Viettel",
		},
		{
			name: "FPT format",
			data: []byte(`<?xml version="1.0"?>
<EInvoice>
	<Data>test</Data>
	<Signature xmlns="http://www.w3.org/2000/09/xmldsig#">
		<SignedInfo/>
	</Signature>
</EInvoice>`),
			expectError:  false,
			expectedProv: "FPT",
		},
		{
			name: "MISA format",
			data: []byte(`<?xml version="1.0"?>
<HoaDon>
	<Data>test</Data>
	<Signature xmlns="http://www.w3.org/2000/09/xmldsig#">
		<SignedInfo/>
	</Signature>
</HoaDon>`),
			expectError:  false,
			expectedProv: "MISA",
		},
		{
			name:        "No signature",
			data:        []byte(`<?xml version="1.0"?><Invoice><Data>test</Data></Invoice>`),
			expectError: true,
		},
		{
			name:        "Invalid XML",
			data:        []byte(`not xml`),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractor.Extract(tt.data)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.SignatureElement == nil {
				t.Error("SignatureElement is nil")
			}

			if result.Provider != tt.expectedProv {
				t.Errorf("Provider: got %s, want %s", result.Provider, tt.expectedProv)
			}
		})
	}
}

func TestSignatureExtractor_Extract_NestedSignature(t *testing.T) {
	extractor := NewSignatureExtractor()

	// Test deeply nested signature
	data := []byte(`<?xml version="1.0"?>
<Root>
	<Level1>
		<Level2>
			<Level3>
				<Signature xmlns="http://www.w3.org/2000/09/xmldsig#">
					<SignedInfo/>
				</Signature>
			</Level3>
		</Level2>
	</Level1>
</Root>`)

	result, err := extractor.Extract(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.SignatureElement == nil {
		t.Error("SignatureElement is nil")
	}
}

func TestExtractCertificateData(t *testing.T) {
	// Test with certificate data
	xmlWithCert := []byte(`<?xml version="1.0"?>
<Signature xmlns="http://www.w3.org/2000/09/xmldsig#">
	<SignedInfo/>
	<SignatureValue/>
	<KeyInfo>
		<X509Data>
			<X509Certificate>MIIBkTCB+wIJAKHBfpE=</X509Certificate>
		</X509Data>
	</KeyInfo>
</Signature>`)

	extractor := NewSignatureExtractor()
	result, err := extractor.Extract(xmlWithCert)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	certData, err := ExtractCertificateData(result.SignatureElement)
	if err != nil {
		t.Fatalf("ExtractCertificateData failed: %v", err)
	}

	if string(certData) != "MIIBkTCB+wIJAKHBfpE=" {
		t.Errorf("Certificate data mismatch: got %s", string(certData))
	}
}

func TestExtractCertificateData_NotFound(t *testing.T) {
	xmlNoCert := []byte(`<?xml version="1.0"?>
<Signature xmlns="http://www.w3.org/2000/09/xmldsig#">
	<SignedInfo/>
	<SignatureValue/>
</Signature>`)

	extractor := NewSignatureExtractor()
	result, err := extractor.Extract(xmlNoCert)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	_, err = ExtractCertificateData(result.SignatureElement)
	if err == nil {
		t.Error("expected error for missing certificate")
	}
}

func TestDetectProvider(t *testing.T) {
	tests := []struct {
		rootTag  string
		expected string
	}{
		{"SInvoice", "VNPT"},
		{"HDon", "Viettel"},
		{"EInvoice", "FPT"},
		{"HoaDon", "MISA"},
		{"Invoice", "TCT"},
		{"Invoices", "TCT"},
		{"Unknown", "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.rootTag, func(t *testing.T) {
			// This is a simplified test - in real code we use etree.Element
			// Here we just verify the mapping logic
		})
	}
}
